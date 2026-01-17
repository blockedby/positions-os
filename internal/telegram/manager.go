package telegram

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/blockedby/positions-os/internal/config"
	"github.com/blockedby/positions-os/internal/logger"
	"github.com/celestix/gotgproto" // Added this import
	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram/auth/qrlogin"
	"gorm.io/gorm"
)

// Status represents the Telegram client status.
type Status string

// Status constants define the possible states of the Telegram client.
const (
	StatusInitializing Status = "INITIALIZING"
	StatusReady        Status = "READY"
	StatusUnauthorized Status = "UNAUTHORIZED"
	StatusError        Status = "ERROR"
)

// ClientFactory is a function that creates a telegram client.
type ClientFactory func(ctx context.Context, cfg *config.Config, db *gorm.DB) (*gotgproto.Client, error)

// QRClientFactory is a function that creates a raw telegram client for QR auth.
type QRClientFactory func(cfg *config.Config) (*QRClientBundle, error)

// ReadReceiptCallback is called when a message read receipt is received.
// peerUserID is the Telegram user ID of the peer who read the message.
// maxMsgID is the highest message ID that was read.
type ReadReceiptCallback func(ctx context.Context, peerUserID int64, maxMsgID int64) error

// Manager handles Telegram client lifecycle and authentication.
type Manager struct {
	client *gotgproto.Client
	db     *gorm.DB
	cfg    *config.Config
	log    *logger.Logger

	status Status
	mu     sync.RWMutex

	clientFactory   ClientFactory
	qrClientFactory QRClientFactory

	// QR flow state management
	qrInProgress atomic.Bool
	qrCancel     context.CancelFunc
	qrMu         sync.Mutex

	// Read receipt callback for dispatcher integration
	readReceiptCallback   ReadReceiptCallback
	readReceiptCallbackMu sync.RWMutex
}

// NewManager creates a new Telegram Manager.
func NewManager(cfg *config.Config, db *gorm.DB) *Manager {
	return &Manager{
		db:              db,
		cfg:             cfg,
		log:             logger.Get(),
		status:          StatusInitializing,
		clientFactory:   NewPersistentClient,
		qrClientFactory: NewQRClient,
	}
}

// SetClientFactory allows overriding the client creation logic (e.g. for testing).
func (m *Manager) SetClientFactory(f ClientFactory) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clientFactory = f
}

// SetQRClientFactory allows overriding the QR client creation logic (e.g. for testing).
func (m *Manager) SetQRClientFactory(f QRClientFactory) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.qrClientFactory = f
}

// SetReadReceiptCallback sets the callback for read receipt events.
// This is used by the dispatcher to automatically update application status to READ.
func (m *Manager) SetReadReceiptCallback(cb ReadReceiptCallback) {
	m.readReceiptCallbackMu.Lock()
	defer m.readReceiptCallbackMu.Unlock()
	m.readReceiptCallback = cb
}

// OnReadReceipt is called by Telegram update handlers when a read receipt is received.
// This forwards the event to the registered callback (if any).
// This method is designed to be called from the Telegram client's update dispatcher.
func (m *Manager) OnReadReceipt(ctx context.Context, peerUserID int64, maxMsgID int64) error {
	m.readReceiptCallbackMu.RLock()
	cb := m.readReceiptCallback
	m.readReceiptCallbackMu.RUnlock()

	if cb == nil {
		return nil // No callback registered, ignore
	}

	return cb(ctx, peerUserID, maxMsgID)
}

// GetStatus returns the current Telegram client status.
func (m *Manager) GetStatus() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

// GetClient returns the underlying Telegram client.
func (m *Manager) GetClient() *gotgproto.Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.client
}

// Init tries to restore session from DB or .env
func (m *Manager) Init(ctx context.Context) error {
	m.mu.Lock()
	m.status = StatusInitializing
	m.mu.Unlock()

	// Check database for existing sessions
	var count int64
	if err := m.db.Table("sessions").Count(&count).Error; err != nil {
		m.log.Warn().Err(err).Msg("telegram: failed to check sessions table")
	}

	// If no session exists, we should not attempt to connect blindly if we want "Silent Start".
	// However, if we want to allow new login, maybe we should?
	// The plan says: "Empty base -> StatusUnauthorized".
	// If we return now, we are Unauthorized.
	if count == 0 {
		m.log.Info().Msg("telegram: no session in database, waiting for auth")
		m.mu.Lock()
		m.status = StatusUnauthorized
		m.mu.Unlock()
		return nil
	}

	client, err := m.clientFactory(ctx, m.cfg, m.db)
	if err != nil {
		m.log.Warn().Err(err).Msg("telegram: failed to initialize persistent client, switching to unauthorized mode")
		m.mu.Lock()
		m.status = StatusUnauthorized
		m.mu.Unlock()
		return nil // Don't return error to keep the app running
	}

	m.mu.Lock()
	m.client = client
	m.status = StatusReady
	m.mu.Unlock()

	m.log.Info().Msg("telegram: client is ready")
	return nil
}

// IsQRInProgress returns true if a QR login flow is currently in progress.
func (m *Manager) IsQRInProgress() bool {
	return m.qrInProgress.Load()
}

// StartQR starts the QR login flow.
// This function blocks until login is successful or context is canceled.
// If a QR flow is already in progress, returns an error immediately.
func (m *Manager) StartQR(ctx context.Context, onQRCode func(url string)) error {
	// Check if already logged in
	m.mu.Lock()
	if m.status == StatusReady {
		m.mu.Unlock()
		return fmt.Errorf("already logged in")
	}
	m.mu.Unlock()

	// Check if QR flow is already in progress
	m.qrMu.Lock()
	if m.qrInProgress.Load() {
		m.qrMu.Unlock()
		m.log.Info().Msg("telegram: QR flow already in progress, ignoring new request")
		return fmt.Errorf("QR login already in progress")
	}

	// Create a cancellable context for this QR flow
	qrCtx, cancel := context.WithCancel(ctx)
	m.qrCancel = cancel
	m.qrInProgress.Store(true)
	m.qrMu.Unlock()

	// Ensure cleanup on exit
	defer func() {
		m.qrInProgress.Store(false)
		m.qrMu.Lock()
		if m.qrCancel != nil {
			m.qrCancel()
			m.qrCancel = nil
		}
		m.qrMu.Unlock()
	}()

	m.log.Info().Time("now", time.Now()).Msg("telegram: starting QR flow, creating QR client")

	// Use the QR client factory (raw td/telegram, not gotgproto)
	bundle, err := m.qrClientFactory(m.cfg)
	if err != nil {
		return fmt.Errorf("create QR client: %w", err)
	}

	var authErr error
	var sessionData *session.Data

	// Run the client connection
	// client.Run blocks until the context is canceled or the function returns
	err = bundle.Client.Run(qrCtx, func(ctx context.Context) error {
		qr := bundle.Client.QR()
		loggedIn := qrlogin.OnLoginToken(&bundle.Dispatcher)

		_, authErr = qr.Auth(ctx, loggedIn, func(_ context.Context, token qrlogin.Token) error {
			m.log.Info().Str("url", token.URL()).Msg("telegram: QR token generated")
			onQRCode(token.URL())
			return nil
		})

		if authErr != nil {
			return authErr
		}

		// On success, capture session
		m.log.Info().Msg("telegram: QR auth success, capturing session")
		loader := session.Loader{Storage: bundle.Storage}
		sessionData, authErr = loader.Load(ctx)
		return authErr
	})

	if err != nil || authErr != nil {
		// If context canceled, it might be normal user cancellation
		if errors.Is(err, context.Canceled) || errors.Is(authErr, context.Canceled) {
			return context.Canceled
		}
		return fmt.Errorf("QR auth flow failed: %w", errors.Join(err, authErr))
	}

	if sessionData == nil {
		return fmt.Errorf("session data is nil after successful auth")
	}

	// Save session to database
	m.log.Info().Msg("telegram: saving session to database")
	if err := m.saveSessionToDB(sessionData); err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	// Reinitialize manager with the new session (this creates the persistent gotgproto client)
	m.log.Info().Msg("telegram: re-initializing manager with new session")
	return m.Init(ctx)
}

// CancelQR cancels any ongoing QR login flow.
func (m *Manager) CancelQR() {
	m.qrMu.Lock()
	defer m.qrMu.Unlock()

	if m.qrCancel != nil {
		m.log.Info().Msg("telegram: canceling ongoing QR flow")
		m.qrCancel()
		m.qrCancel = nil
	}
	m.qrInProgress.Store(false)
}

func (m *Manager) saveSessionToDB(data *session.Data) error {
	// Convert to gotgproto compatible session
	sess, err := ConvertToGotgprotoSession(data)
	if err != nil {
		return err
	}

	// Save to database
	// We might need to handle existing sessions (e.g. drop table logic) if we want clean state,
	// but normally overwriting or just saving is fine if ID logic is handled.
	// storage.Session has Version as primary key? No, in guide:
	// type Session struct { Version int `gorm:"primary_key"`; Data []byte }
	// So we might need to delete old one or upsert.
	// Since primary key is Version (fixed to 1), Save should upsert.

	return m.db.Save(sess).Error
}

// Stop stops the Telegram client.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.client != nil {
		m.client.Stop()
	}
}
