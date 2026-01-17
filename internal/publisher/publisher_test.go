package publisher

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/blockedby/positions-os/internal/collector"
)

// MockNATSClient mocks the nats client operations we need
type MockNATSClient struct {
	PublishedSubject string
	PublishedData    []byte
	PublishError     error
}

func (m *MockNATSClient) Publish(subject string, data []byte) error {
	m.PublishedSubject = subject
	m.PublishedData = data
	return m.PublishError
}

func TestNATSPublisher_PublishJobNew(t *testing.T) {
	mock := &MockNATSClient{}
	pub := &NATSPublisher{
		js: mock,
	}

	event := collector.JobNewEvent{
		JobID:      uuid.New(),
		TargetID:   uuid.New(),
		ExternalID: "123",
		RawContent: "test",
		CreatedAt:  time.Now(),
	}

	err := pub.PublishJobNew(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.PublishedSubject != "jobs.new" {
		t.Errorf("subject = %s, want jobs.new", mock.PublishedSubject)
	}

	if len(mock.PublishedData) == 0 {
		t.Error("payload should not be empty")
	}
}
