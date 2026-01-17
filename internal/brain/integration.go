package brain

import (
	"context"
	"fmt"

	"github.com/blockedby/positions-os/internal/logger"
	"github.com/google/uuid"
)

// JobRepository extends Repository with methods needed for service integration.
type JobRepository interface {
	Repository
	GetJobData(id uuid.UUID) (map[string]string, error)
}

// PrepareService wraps Service to implement BrainService interface.
// It fetches job data and calls the full tailoring pipeline.
type PrepareService struct {
	service *Service
	repo    JobRepository
}

// NewPrepareService creates a new prepare service.
func NewPrepareService(svc *Service, repo JobRepository) *PrepareService {
	return &PrepareService{
		service: svc,
		repo:    repo,
	}
}

// PrepareJob runs the full tailoring pipeline for a job.
// Implements BrainService interface.
func (p *PrepareService) PrepareJob(jobID string) (*PipelineResult, error) {
	logger.Info("preparing job: " + jobID)

	id, err := uuid.Parse(jobID)
	if err != nil {
		return nil, fmt.Errorf("invalid job ID: %w", err)
	}

	// Fetch job data from repository
	jobData, err := p.repo.GetJobData(id)
	if err != nil {
		logger.Error("failed to get job data", err)
		return nil, fmt.Errorf("get job data: %w", err)
	}

	// Run the full pipeline
	result, err := p.service.TailorResumePipeline(context.Background(), jobID, jobData)
	if err != nil {
		logger.Error("pipeline failed", err)
		return nil, err
	}

	// Update job outputs
	if err := p.repo.UpdateBrainOutputs(id, result.ResumePDFPath, result.CoverLetterMD); err != nil {
		logger.Error("failed to update job outputs", err)
		// Non-fatal error, pipeline succeeded
	}

	logger.Info("job prepared: " + jobID)
	return result, nil
}

// InMemoryRepository is a simple in-memory repository for testing.
type InMemoryRepository struct {
	jobs map[uuid.UUID]*BrainJob
}

// NewInMemoryRepository creates a new in-memory repository.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		jobs: make(map[uuid.UUID]*BrainJob),
	}
}

// AddJob adds a job to the repository.
func (m *InMemoryRepository) AddJob(job *BrainJob) {
	m.jobs[job.ID] = job
}

// GetByID implements Repository.
func (m *InMemoryRepository) GetByID(id uuid.UUID) (*BrainJob, error) {
	job, ok := m.jobs[id]
	if !ok {
		return nil, ErrJobNotFound
	}
	return job, nil
}

// UpdateBrainOutputs implements Repository.
func (m *InMemoryRepository) UpdateBrainOutputs(id uuid.UUID, resumePath, coverText string) error {
	job, ok := m.jobs[id]
	if !ok {
		return ErrJobNotFound
	}
	job.TailoredResumePath = resumePath
	job.CoverLetterText = coverText
	return nil
}

// GetJobData implements JobRepository.
func (m *InMemoryRepository) GetJobData(id uuid.UUID) (map[string]string, error) {
	job, ok := m.jobs[id]
	if !ok {
		return nil, ErrJobNotFound
	}
	return job.StructuredData, nil
}
