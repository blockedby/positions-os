package repository

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// test message id filter for deduplication using min/max range
func TestMessageIDFilter_FilterNew(t *testing.T) {
	tests := []struct {
		name        string
		minParsed   int64
		maxParsed   int64
		inputIDs    []int64
		expectedIDs []int64
	}{
		{
			name:        "all new when no parsed",
			minParsed:   0,
			maxParsed:   0,
			inputIDs:    []int64{100, 101, 102},
			expectedIDs: []int64{100, 101, 102},
		},
		{
			name:        "filters out messages within range",
			minParsed:   100,
			maxParsed:   200,
			inputIDs:    []int64{99, 100, 150, 200, 201},
			expectedIDs: []int64{99, 201},
		},
		{
			name:        "returns empty when all within range",
			minParsed:   50,
			maxParsed:   200,
			inputIDs:    []int64{99, 100, 101},
			expectedIDs: []int64{},
		},
		{
			name:        "handles empty input",
			minParsed:   100,
			maxParsed:   200,
			inputIDs:    []int64{},
			expectedIDs: []int64{},
		},
		{
			name:        "handles nil input",
			minParsed:   100,
			maxParsed:   200,
			inputIDs:    nil,
			expectedIDs: []int64{},
		},
		{
			name:        "boundary case - exactly at min",
			minParsed:   100,
			maxParsed:   200,
			inputIDs:    []int64{100},
			expectedIDs: []int64{},
		},
		{
			name:        "boundary case - exactly at max",
			minParsed:   100,
			maxParsed:   200,
			inputIDs:    []int64{200},
			expectedIDs: []int64{},
		},
		{
			name:        "boundary case - just below min",
			minParsed:   100,
			maxParsed:   200,
			inputIDs:    []int64{99},
			expectedIDs: []int64{99},
		},
		{
			name:        "boundary case - just above max",
			minParsed:   100,
			maxParsed:   200,
			inputIDs:    []int64{201},
			expectedIDs: []int64{201},
		},
		{
			name:        "scraping older messages scenario",
			minParsed:   1896,
			maxParsed:   1932,
			inputIDs:    []int64{1828, 1850, 1896, 1900, 1932},
			expectedIDs: []int64{1828, 1850},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewMessageIDFilter(tt.minParsed, tt.maxParsed)
			result := filter.FilterNew(tt.inputIDs)

			// handle nil vs empty slice comparison
			if len(tt.expectedIDs) == 0 && len(result) == 0 {
				return // both empty, test passes
			}

			if len(result) != len(tt.expectedIDs) {
				t.Errorf("FilterNew() returned %d items, want %d", len(result), len(tt.expectedIDs))
				return
			}

			for i, id := range result {
				if id != tt.expectedIDs[i] {
					t.Errorf("FilterNew()[%d] = %d, want %d", i, id, tt.expectedIDs[i])
				}
			}
		})
	}
}

// test range contains check
func TestParsedRange_Contains(t *testing.T) {
	tests := []struct {
		name     string
		minID    int64
		maxID    int64
		checkID  int64
		expected bool
	}{
		{
			name:     "id within range",
			minID:    100,
			maxID:    200,
			checkID:  150,
			expected: true,
		},
		{
			name:     "id at min boundary",
			minID:    100,
			maxID:    200,
			checkID:  100,
			expected: true,
		},
		{
			name:     "id at max boundary",
			minID:    100,
			maxID:    200,
			checkID:  200,
			expected: true,
		},
		{
			name:     "id below range",
			minID:    100,
			maxID:    200,
			checkID:  50,
			expected: false,
		},
		{
			name:     "id above range",
			minID:    100,
			maxID:    200,
			checkID:  250,
			expected: false,
		},
		{
			name:     "empty range",
			minID:    0,
			maxID:    0,
			checkID:  0,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ParsedRange{
				MinMsgID: tt.minID,
				MaxMsgID: tt.maxID,
			}
			if got := r.Contains(tt.checkID); got != tt.expected {
				t.Errorf("Contains(%d) = %v, want %v", tt.checkID, got, tt.expected)
			}
		})
	}
}

// test range update logic
func TestParsedRange_Extend(t *testing.T) {
	tests := []struct {
		name    string
		initial *ParsedRange
		newMin  int64
		newMax  int64
		wantMin int64
		wantMax int64
	}{
		{
			name:    "extend upward",
			initial: &ParsedRange{MinMsgID: 100, MaxMsgID: 200},
			newMin:  150,
			newMax:  300,
			wantMin: 100,
			wantMax: 300,
		},
		{
			name:    "extend downward",
			initial: &ParsedRange{MinMsgID: 100, MaxMsgID: 200},
			newMin:  50,
			newMax:  150,
			wantMin: 50,
			wantMax: 200,
		},
		{
			name:    "extend both directions",
			initial: &ParsedRange{MinMsgID: 100, MaxMsgID: 200},
			newMin:  50,
			newMax:  300,
			wantMin: 50,
			wantMax: 300,
		},
		{
			name:    "no extension needed",
			initial: &ParsedRange{MinMsgID: 50, MaxMsgID: 300},
			newMin:  100,
			newMax:  200,
			wantMin: 50,
			wantMax: 300,
		},
		{
			name:    "first range creation",
			initial: &ParsedRange{MinMsgID: 0, MaxMsgID: 0},
			newMin:  100,
			newMax:  200,
			wantMin: 100,
			wantMax: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.initial.Extend(tt.newMin, tt.newMax)
			if tt.initial.MinMsgID != tt.wantMin {
				t.Errorf("MinMsgID = %d, want %d", tt.initial.MinMsgID, tt.wantMin)
			}
			if tt.initial.MaxMsgID != tt.wantMax {
				t.Errorf("MaxMsgID = %d, want %d", tt.initial.MaxMsgID, tt.wantMax)
			}
		})
	}
}

// test smart message filter that checks both range AND existing jobs
func TestSmartMessageFilter_FilterNew(t *testing.T) {
	tests := []struct {
		name         string
		minParsed    int64
		maxParsed    int64
		existingJobs []int64
		inputIDs     []int64
		expectedIDs  []int64
	}{
		{
			name:         "all new when no parsed range",
			minParsed:    0,
			maxParsed:    0,
			existingJobs: []int64{},
			inputIDs:     []int64{100, 101, 102},
			expectedIDs:  []int64{100, 101, 102},
		},
		{
			name:         "filters only messages with existing jobs",
			minParsed:    100,
			maxParsed:    110,
			existingJobs: []int64{105, 106, 107},
			inputIDs:     []int64{103, 105, 106, 107, 108},
			expectedIDs:  []int64{103, 108},
		},
		{
			name:         "messages outside range are always new",
			minParsed:    100,
			maxParsed:    110,
			existingJobs: []int64{105},
			inputIDs:     []int64{99, 105, 111},
			expectedIDs:  []int64{99, 111},
		},
		{
			name:         "empty existing jobs means all in-range are new",
			minParsed:    100,
			maxParsed:    110,
			existingJobs: []int64{},
			inputIDs:     []int64{103, 105, 107},
			expectedIDs:  []int64{103, 105, 107},
		},
		{
			name:         "real scenario - gaps in job creation",
			minParsed:    1911,
			maxParsed:    1933,
			existingJobs: []int64{1928, 1929, 1930, 1931, 1932, 1933},
			inputIDs:     []int64{1920, 1925, 1928, 1930, 1933, 1934},
			expectedIDs:  []int64{1920, 1925, 1934},
		},
		{
			name:         "handles empty input",
			minParsed:    100,
			maxParsed:    200,
			existingJobs: []int64{150},
			inputIDs:     []int64{},
			expectedIDs:  []int64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewSmartMessageFilter(tt.minParsed, tt.maxParsed, tt.existingJobs)
			result := filter.FilterNew(tt.inputIDs)

			if len(tt.expectedIDs) == 0 && len(result) == 0 {
				return
			}

			if len(result) != len(tt.expectedIDs) {
				t.Errorf("FilterNew() returned %d items, want %d\ngot: %v\nwant: %v",
					len(result), len(tt.expectedIDs), result, tt.expectedIDs)
				return
			}

			for i, id := range result {
				if id != tt.expectedIDs[i] {
					t.Errorf("FilterNew()[%d] = %d, want %d", i, id, tt.expectedIDs[i])
				}
			}
		})
	}
}

// TestRangesRepository_NewSmartFilter_Interface verifies the method signature exists
func TestRangesRepository_NewSmartFilter_Interface(t *testing.T) {
	var _ interface {
		NewSmartFilter(ctx context.Context, targetID uuid.UUID, existingJobIDs []int64) (*SmartMessageFilter, error)
	} = (*RangesRepository)(nil)
}
