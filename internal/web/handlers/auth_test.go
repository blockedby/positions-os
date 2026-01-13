package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/blockedby/positions-os/internal/telegram"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTelegramClient is a mock implementation of the TelegramClient interface
type MockTelegramClient struct {
	mock.Mock
}

func (m *MockTelegramClient) StartQR(ctx context.Context, onQRCode func(url string)) error {
	args := m.Called(ctx, onQRCode)
	return args.Error(0)
}

func (m *MockTelegramClient) GetStatus() telegram.Status {
	args := m.Called()
	return args.Get(0).(telegram.Status)
}

func TestAuthHandler_StartQR_Success(t *testing.T) {
	// Setup
	mockClient := new(MockTelegramClient)

	// Expect StartQR to be called. We return nil error.
	// It should block? If it blocks, the handler better not block the HTTP request forever.
	// Actually, usually the HTTP request initiates the flow.
	// If StartQR blocks until *success*, then the HTTP handler should probably NOT wait for it to finish,
	// but rather spawn it and return success "Flow started".
	// The QR code comes via WebSocket.

	mockClient.On("StartQR", mock.Anything, mock.Anything).Return(nil)
	mockClient.On("GetStatus").Return(telegram.StatusUnauthorized)

	h := NewAuthHandler(mockClient, nil) // Hub is nil for now

	req, _ := http.NewRequest("POST", "/api/v1/auth/qr", nil)
	rr := httptest.NewRecorder()

	// Act
	h.StartQR(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, `{"status":"started"}`, rr.Body.String())

	// Verify StartQR was called (likely asynchronously if we design it right)
	// Wait a bit if it's async
	time.Sleep(100 * time.Millisecond)
	mockClient.AssertExpectations(t)
}

func TestAuthHandler_StartQR_AlreadyLoggedIn(t *testing.T) {
	// Setup
	mockClient := new(MockTelegramClient)

	mockClient.On("GetStatus").Return(telegram.StatusReady)

	h := NewAuthHandler(mockClient, nil)

	req, _ := http.NewRequest("POST", "/api/v1/auth/qr", nil)
	rr := httptest.NewRecorder()

	// Act
	h.StartQR(rr, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.JSONEq(t, `{"error":"already logged in"}`, rr.Body.String())
}

type MockHub struct {
	mock.Mock
}

func (m *MockHub) Broadcast(message interface{}) {
	m.Called(message)
}

func TestAuthHandler_StartQR_BroadcastsQRCode(t *testing.T) {
	// Setup
	mockClient := new(MockTelegramClient)
	mockHub := new(MockHub)

	mockClient.On("GetStatus").Return(telegram.StatusUnauthorized)

	h := NewAuthHandler(mockClient, mockHub)

	// Single expectation with Run to trigger callback
	mockClient.On("StartQR", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		onQRCode := args.Get(1).(func(string))
		onQRCode("http://t.me/auth/test_token")
	}).Return(nil)

	// Expect Broadcast to be called with the QR code
	mockHub.On("Broadcast", mock.MatchedBy(func(msg interface{}) bool {
		m, ok := msg.(map[string]string)
		if !ok {
			return false
		}
		return m["type"] == "tg_qr" && m["url"] == "http://t.me/auth/test_token"
	})).Return()

	req, _ := http.NewRequest("POST", "/api/v1/auth/qr", nil)
	rr := httptest.NewRecorder()

	// Act
	h.StartQR(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)

	// Wait for async broadcast
	time.Sleep(50 * time.Millisecond)
	mockHub.AssertExpectations(t)
}

func TestAuthHandler_StartQR_BroadcastsError(t *testing.T) {
	// Setup
	mockClient := new(MockTelegramClient)
	mockHub := new(MockHub)

	mockClient.On("GetStatus").Return(telegram.StatusUnauthorized)
	
	h := NewAuthHandler(mockClient, mockHub)

	// Mock StartQR to return an error
	mockClient.On("StartQR", mock.Anything, mock.Anything).Return(errors.New("auth failed"))

	// Expect Broadcast to be called with the error
	mockHub.On("Broadcast", mock.MatchedBy(func(msg interface{}) bool {
		m, ok := msg.(map[string]string)
		if !ok {
			return false
		}
		return m["type"] == "error" && m["message"] == "auth failed"
	})).Return()

	req, _ := http.NewRequest("POST", "/api/v1/auth/qr", nil)
	rr := httptest.NewRecorder()

	// Act
	h.StartQR(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)

	// Wait for async broadcast
	time.Sleep(50 * time.Millisecond)
	mockHub.AssertExpectations(t)
}

func TestAuthHandler_StartQR_ConcurrentCalls(t *testing.T) {
	mockClient := new(MockTelegramClient)
	mockHub := new(MockHub)

	mockClient.On("GetStatus").Return(telegram.StatusUnauthorized)
	mockClient.On("StartQR", mock.Anything, mock.Anything).Return(nil)

	h := NewAuthHandler(mockClient, mockHub)

	req, _ := http.NewRequest("POST", "/api/v1/auth/qr", nil)

	// Start 10 concurrent requests
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rr := httptest.NewRecorder()
			h.StartQR(rr, req)
			assert.Equal(t, http.StatusOK, rr.Code)
		}()
	}
	wg.Wait()
}

// TestAuthHandler_StartQR_BroadcastsSuccessOnSuccessfulAuth tests that
// when auth completes successfully (StartQR returns nil), a "tg_auth_success"
// message is broadcast to hide the QR code on the frontend.
func TestAuthHandler_StartQR_BroadcastsSuccessOnSuccessfulAuth(t *testing.T) {
	// Setup
	mockClient := new(MockTelegramClient)
	mockHub := new(MockHub)

	mockClient.On("GetStatus").Return(telegram.StatusUnauthorized)

	h := NewAuthHandler(mockClient, mockHub)

	// Mock StartQR to return nil (success)
	mockClient.On("StartQR", mock.Anything, mock.Anything).Return(nil)

	// Expect success broadcast when auth completes
	mockHub.On("Broadcast", mock.MatchedBy(func(msg interface{}) bool {
		m, ok := msg.(map[string]string)
		if !ok {
			return false
		}
		return m["type"] == "tg_auth_success"
	})).Return()

	req, _ := http.NewRequest("POST", "/api/v1/auth/qr", nil)
	rr := httptest.NewRecorder()

	// Act
	h.StartQR(rr, req)

	// Assert HTTP response is immediate
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, `{"status":"started"}`, rr.Body.String())

	// Wait for async goroutine to complete and broadcast success
	time.Sleep(100 * time.Millisecond)

	// Assert success was broadcast
	mockHub.AssertExpectations(t)
	mockClient.AssertExpectations(t)
}

// TestAuthHandler_StartQR_QRHiddenAfterSuccess tests the full flow:
// 1. QR appears (tg_qr)
// 2. Auth succeeds (tg_auth_success)
// This ensures the QR code would be hidden on the frontend.
func TestAuthHandler_StartQR_QRHiddenAfterSuccess(t *testing.T) {
	// Setup
	mockClient := new(MockTelegramClient)
	mockHub := new(MockHub)

	mockClient.On("GetStatus").Return(telegram.StatusUnauthorized)

	h := NewAuthHandler(mockClient, mockHub)

	// Track broadcast order
	var broadcasts []interface{}
	mockHub.On("Broadcast", mock.Anything).Run(func(args mock.Arguments) {
		broadcasts = append(broadcasts, args.Get(0))
	}).Return()

	// Mock StartQR to trigger QR callback, then return success
	mockClient.On("StartQR", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		onQRCode := args.Get(1).(func(string))
		// Simulate QR being sent
		onQRCode("tg://login?token=test123")
	}).Return(nil)

	req, _ := http.NewRequest("POST", "/api/v1/auth/qr", nil)
	rr := httptest.NewRecorder()

	// Act
	h.StartQR(rr, req)

	// Wait for async goroutine
	time.Sleep(100 * time.Millisecond)

	// Assert: we should have 2 broadcasts
	assert.Equal(t, 2, len(broadcasts), "expected 2 broadcasts: qr + success")

	// First is QR code
	qrMsg, ok := broadcasts[0].(map[string]string)
	assert.True(t, ok, "first broadcast should be map[string]string")
	assert.Equal(t, "tg_qr", qrMsg["type"])
	assert.Equal(t, "tg://login?token=test123", qrMsg["url"])

	// Second is success
	successMsg, ok := broadcasts[1].(map[string]string)
	assert.True(t, ok, "second broadcast should be map[string]string")
	assert.Equal(t, "tg_auth_success", successMsg["type"])
}
