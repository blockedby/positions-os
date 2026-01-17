package api

import (
	"context"

	"github.com/blockedby/positions-os/internal/dispatcher"
	"github.com/blockedby/positions-os/internal/models"
	"github.com/blockedby/positions-os/internal/repository"
	"github.com/blockedby/positions-os/internal/telegram"
	"github.com/google/uuid"
)

// JobsRepository defines the interface for job data access.
type JobsRepository interface {
	List(ctx context.Context, filter repository.JobFilter) ([]*repository.Job, int, error)
	GetByID(ctx context.Context, id uuid.UUID) (*repository.Job, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
	BulkDelete(ctx context.Context, ids []uuid.UUID) (int, error)
}

// TargetsRepository defines the interface for target data access.
type TargetsRepository interface {
	List(ctx context.Context) ([]repository.ScrapingTarget, error)
	Create(ctx context.Context, t *repository.ScrapingTarget) error
	GetByID(ctx context.Context, id uuid.UUID) (*repository.ScrapingTarget, error)
	Update(ctx context.Context, t *repository.ScrapingTarget) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// StatsRepository defines the interface for stats data access.
type StatsRepository interface {
	GetStats(ctx context.Context) (*repository.DashboardStats, error)
}

// ApplicationsRepository defines the interface for application data access.
type ApplicationsRepository interface {
	Create(ctx context.Context, app *models.JobApplication) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.JobApplication, error)
	GetByJobID(ctx context.Context, jobID uuid.UUID) ([]*models.JobApplication, error)
	UpdateDeliveryStatus(ctx context.Context, id uuid.UUID, status models.DeliveryStatus) error
}

// TelegramClient defines the interface for Telegram operations.
type TelegramClient interface {
	GetStatus() telegram.Status
	IsQRInProgress() bool
	StartQR(ctx context.Context, onURL func(string)) error
}

// CollectorService defines the interface for scraping operations.
type CollectorService interface {
	StartScrape(ctx context.Context, channel string, limit int, topicIDs []int64) error
	StopScrape() error
	IsRunning() bool
	Status() (target string, progress int, newJobs int)
}

// DispatcherService defines the interface for sending applications.
type DispatcherService interface {
	SendApplication(ctx context.Context, req *dispatcher.SendRequest) error
}

// HubBroadcaster defines the interface for WebSocket broadcasting.
type HubBroadcaster interface {
	Broadcast(message interface{})
}
