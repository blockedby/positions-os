package collector

import (
	"testing"
)

// test scrape request validation
func TestScrapeRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     ScrapeRequest
		wantErr error
	}{
		{
			name:    "empty request - requires source",
			req:     ScrapeRequest{},
			wantErr: ErrChannelRequired,
		},
		{
			name: "valid channel only",
			req: ScrapeRequest{
				Channel: "@golang_jobs",
			},
			wantErr: nil,
		},
		{
			name: "valid channel without @",
			req: ScrapeRequest{
				Channel: "golang_jobs",
			},
			wantErr: nil,
		},
		{
			name: "valid with limit",
			req: ScrapeRequest{
				Channel: "@test",
				Limit:   100,
			},
			wantErr: nil,
		},
		{
			name: "negative limit",
			req: ScrapeRequest{
				Channel: "@test",
				Limit:   -1,
			},
			wantErr: ErrInvalidLimit,
		},
		{
			name: "valid date format",
			req: ScrapeRequest{
				Channel: "@test",
				Until:   "2024-01-15",
			},
			wantErr: nil,
		},
		{
			name: "invalid date format",
			req: ScrapeRequest{
				Channel: "@test",
				Until:   "not-a-date",
			},
			wantErr: ErrInvalidDate,
		},
		{
			name: "invalid date format - wrong order",
			req: ScrapeRequest{
				Channel: "@test",
				Until:   "15-01-2024",
			},
			wantErr: ErrInvalidDate,
		},
		{
			name: "future date",
			req: ScrapeRequest{
				Channel: "@test",
				Until:   "2099-12-31",
			},
			wantErr: ErrFutureDate,
		},
		{
			name: "topic ids without forum",
			req: ScrapeRequest{
				Channel:  "@test",
				TopicIDs: []int{1, 15, 28},
			},
			wantErr: nil, // validation at runtime, not in basic validate
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
				return
			}
			if err == nil {
				t.Errorf("Validate() expected error %v, got nil", tt.wantErr)
				return
			}
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// test until date parsing
func TestScrapeRequest_UntilTime(t *testing.T) {
	tests := []struct {
		name     string
		until    string
		wantNil  bool
		wantYear int
	}{
		{
			name:    "empty until",
			until:   "",
			wantNil: true,
		},
		{
			name:     "valid date",
			until:    "2024-06-15",
			wantNil:  false,
			wantYear: 2024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := ScrapeRequest{Channel: "@test", Until: tt.until}
			result := req.UntilTime()
			if tt.wantNil {
				if result != nil {
					t.Error("UntilTime() should return nil")
				}
				return
			}
			if result == nil {
				t.Error("UntilTime() should not return nil")
				return
			}
			if result.Year() != tt.wantYear {
				t.Errorf("UntilTime().Year() = %d, want %d", result.Year(), tt.wantYear)
			}
		})
	}
}
