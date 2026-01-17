package handlers

import (
	"context"

	"github.com/google/uuid"

	"github.com/blockedby/positions-os/internal/repository"
)

// JobsRepository defines interface for jobs data access
type JobsRepository interface {
	List(ctx context.Context, filter repository.JobFilter) ([]*repository.Job, int, error)
	GetByID(ctx context.Context, id uuid.UUID) (*repository.Job, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
}

// StatsRepository defines interface for stats data access
type StatsRepository interface {
	GetStats(ctx context.Context) (*repository.DashboardStats, error)
}
