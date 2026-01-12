package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DashboardStats struct {
	TotalJobs      int `json:"total_jobs"`
	AnalyzedJobs   int `json:"analyzed_jobs"`
	InterestedJobs int `json:"interested_jobs"`
	RejectedJobs   int `json:"rejected_jobs"`
	TodayJobs      int `json:"today_jobs"`
	ActiveTargets  int `json:"active_targets"`
}

type StatsRepository struct {
	pool *pgxpool.Pool
}

func NewStatsRepository(pool *pgxpool.Pool) *StatsRepository {
	return &StatsRepository{pool: pool}
}

func (r *StatsRepository) GetStats(ctx context.Context) (*DashboardStats, error) {
	stats := &DashboardStats{}

	// Aggregated query for jobs
	err := r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'ANALYZED' THEN 1 END) as analyzed,
			COUNT(CASE WHEN status = 'INTERESTED' THEN 1 END) as interested,
			COUNT(CASE WHEN status = 'REJECTED' THEN 1 END) as rejected,
			COUNT(CASE WHEN created_at >= CURRENT_DATE THEN 1 END) as today
		FROM jobs
	`).Scan(&stats.TotalJobs, &stats.AnalyzedJobs, &stats.InterestedJobs, &stats.RejectedJobs, &stats.TodayJobs)
	if err != nil {
		return nil, fmt.Errorf("get job stats: %w", err)
	}

	// Active targets
	err = r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM scraping_targets WHERE is_active = true
	`).Scan(&stats.ActiveTargets)
	if err != nil {
		return nil, fmt.Errorf("get target stats: %w", err)
	}

	return stats, nil
}
