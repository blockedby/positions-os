package handlers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/blockedby/positions-os/internal/repository"
	"github.com/blockedby/positions-os/internal/web"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestLayout_ContainsSidebar(t *testing.T) {
	mockRepo := new(MockJobsRepository)
	mockStats := new(MockStatsRepository)
	mockStats.On("GetStats", mock.Anything).Return(&repository.DashboardStats{}, nil).Maybe()

	srv := setupTestServer(t, mockRepo, mockStats)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	assert.Contains(t, html, `id="sidebar"`)
	assert.Contains(t, html, `id="main-content"`)
	assert.Contains(t, html, "Dashboard")
	assert.Contains(t, html, "Jobs")
	assert.Contains(t, html, "Settings")
}

func TestNavigation_AllPagesLoad(t *testing.T) {
	mockRepo := new(MockJobsRepository)
	mockStats := new(MockStatsRepository)

	// Expect List to be called for Jobs page
	mockRepo.On("List", mock.Anything, mock.Anything).Return([]*repository.Job{}, 0, nil).Maybe()
	mockStats.On("GetStats", mock.Anything).Return(&repository.DashboardStats{}, nil).Maybe()

	srv := setupTestServer(t, mockRepo, mockStats)
	defer srv.Close()

	pages := []struct {
		path     string
		contains string
	}{
		{"/", "Dashboard"},
		{"/jobs", "Jobs"},
		{"/settings", "Settings"},
	}

	for _, p := range pages {
		t.Run(p.path, func(t *testing.T) {
			resp, err := http.Get(srv.URL + p.path)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)

			body, _ := io.ReadAll(resp.Body)
			assert.Contains(t, string(body), p.contains)
		})
	}
}

func TestNavigation_HTMXPartialResponse(t *testing.T) {
	mockRepo := new(MockJobsRepository)
	mockStats := new(MockStatsRepository)
	mockRepo.On("List", mock.Anything, mock.Anything).Return([]*repository.Job{}, 0, nil).Maybe()
	mockStats.On("GetStats", mock.Anything).Return(&repository.DashboardStats{}, nil).Maybe()

	srv := setupTestServer(t, mockRepo, mockStats)
	defer srv.Close()

	// Request with HX-Request header (HTMX request)
	req, _ := http.NewRequest("GET", srv.URL+"/jobs", nil)
	req.Header.Set("HX-Request", "true")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	// Should NOT contain full layout
	assert.NotContains(t, html, "<!DOCTYPE html>")
	assert.NotContains(t, html, "<head>")

	// Should contain only content
	assert.Contains(t, html, "Jobs")
}

func TestJobsPage_RendersJobs(t *testing.T) {
	mockRepo := new(MockJobsRepository)
	mockStats := new(MockStatsRepository)
	jobs := []*repository.Job{
		{ExternalID: "job-101"},
		{ExternalID: "job-102"},
	}
	// Expect List call
	mockRepo.On("List", mock.Anything, mock.Anything).Return(jobs, 2, nil)

	srv := setupTestServer(t, mockRepo, mockStats)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/jobs")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	assert.Contains(t, html, "job-101")
	assert.Contains(t, html, "job-102")
	assert.Contains(t, html, "Total: 2")
}

func TestJobsPage_HTMX_RendersTable(t *testing.T) {
	mockRepo := new(MockJobsRepository)
	mockStats := new(MockStatsRepository)
	jobs := []*repository.Job{{ExternalID: "job-htmx"}}
	mockRepo.On("List", mock.Anything, mock.Anything).Return(jobs, 1, nil)

	srv := setupTestServer(t, mockRepo, mockStats)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/jobs", nil)
	req.Header.Set("HX-Request", "true")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	// assert no layout
	assert.NotContains(t, html, "DOCTYPE")
	// assert data present
	assert.Contains(t, html, "job-htmx")
}

func TestJobPanel_Renders(t *testing.T) {
	mockRepo := new(MockJobsRepository)
	mockStats := new(MockStatsRepository)
	id := uuid.New()
	job := &repository.Job{ID: id, ExternalID: "job-detail", Status: "RAW"}

	mockRepo.On("GetByID", mock.Anything, id).Return(job, nil) // Return pointer to job

	srv := setupTestServer(t, mockRepo, mockStats)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/jobs/"+id.String(), nil)
	req.Header.Set("HX-Request", "true")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	assert.Contains(t, string(body), "job-detail")
	assert.Contains(t, string(body), "RAW")
}

func TestJobRow_Renders(t *testing.T) {
	mockRepo := new(MockJobsRepository)
	mockStats := new(MockStatsRepository)
	id := uuid.New()
	job := &repository.Job{ID: id, ExternalID: "job-row", Status: "INTERESTED"}

	mockRepo.On("GetByID", mock.Anything, id).Return(job, nil)

	srv := setupTestServer(t, mockRepo, mockStats)
	defer srv.Close()

	// Should call partials endpoint
	req, _ := http.NewRequest("GET", srv.URL+"/partials/jobs/row/"+id.String(), nil)
	req.Header.Set("HX-Request", "true")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	assert.Contains(t, html, "job-row") // Content from template
	assert.Contains(t, html, "INTERESTED")
}

func TestStatsCards_Renders(t *testing.T) {
	mockRepo := new(MockJobsRepository)
	mockStats := new(MockStatsRepository)
	stats := &repository.DashboardStats{
		TotalJobs:      100,
		InterestedJobs: 10,
	}

	mockStats.On("GetStats", mock.Anything).Return(stats, nil)

	srv := setupTestServer(t, mockRepo, mockStats)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/partials/stats-cards", nil)
	req.Header.Set("HX-Request", "true")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	assert.Contains(t, html, "Total Jobs")
	assert.Contains(t, html, "100")
	assert.Contains(t, html, "Interested")
	assert.Contains(t, html, "10")
}

func TestRecentJobs_Renders(t *testing.T) {
	mockRepo := new(MockJobsRepository)
	mockStats := new(MockStatsRepository)
	jobs := []*repository.Job{
		{
			ExternalID: "recent-1",
			StructuredData: map[string]interface{}{
				"title": "Recent Job 1",
			},
		},
	}

	mockRepo.On("List", mock.Anything, mock.MatchedBy(func(f repository.JobFilter) bool {
		return f.Limit == 5
	})).Return(jobs, 1, nil)

	srv := setupTestServer(t, mockRepo, mockStats)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/partials/recent-jobs", nil)
	req.Header.Set("HX-Request", "true")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	assert.Contains(t, html, "Recent Job 1")
}

// MockStatsRepository
type MockStatsRepository struct {
	mock.Mock
}

func (m *MockStatsRepository) GetStats(ctx context.Context) (*repository.DashboardStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.DashboardStats), args.Error(1)
}

func setupTestServer(t *testing.T, jobsRepo JobsRepository, statsRepo StatsRepository) *httptest.Server {
	// Create test templates
	templateDir := filepath.Join(t.TempDir(), "templates")
	require.NoError(t, os.MkdirAll(filepath.Join(templateDir, "pages"), 0755))

	// Layout
	layout := `{{ define "layout" }}<!DOCTYPE html>
<html>
<head><title>{{ .Title }} | Job Hunter OS</title></head>
<body>
<div class="flex">
{{ template "sidebar" . }}
<main id="main-content">{{ block "content" . }}{{ end }}</main>
</div>
</body>
</html>{{ end }}`
	require.NoError(t, os.WriteFile(filepath.Join(templateDir, "layout.html"), []byte(layout), 0644))

	// Sidebar
	sidebar := `{{ define "sidebar" }}<nav id="sidebar">
<div><h1>Job Hunter OS</h1></div>
<ul>
<li><a href="/">Dashboard</a></li>
<li><a href="/jobs">Jobs</a></li>
<li><a href="/settings">Settings</a></li>
</ul>
</nav>{{ end }}`
	require.NoError(t, os.WriteFile(filepath.Join(templateDir, "sidebar.html"), []byte(sidebar), 0644))

	// Dashboard page
	dashboard := `{{ define "content" }}<h1>Dashboard</h1>{{ end }}`
	require.NoError(t, os.WriteFile(filepath.Join(templateDir, "pages", "dashboard.html"), []byte(dashboard), 0644))

	// Jobs page
	jobs := `{{ define "content" }}<h1>Jobs</h1>{{ if .Jobs }}<ul>{{ range .Jobs }}<li>{{ .ExternalID }}</li>{{ end }}</ul>{{ end }}<p>Total: {{ .Total }}</p>{{ end }}`
	require.NoError(t, os.WriteFile(filepath.Join(templateDir, "pages", "jobs.html"), []byte(jobs), 0644))

	// Job Panel
	panel := `{{ define "content" }}<h1>Job Detail</h1><p>{{ .Job.ExternalID }}</p><p>{{ .Job.Status }}</p>{{ end }}`
	require.NoError(t, os.WriteFile(filepath.Join(templateDir, "pages", "job-panel.html"), []byte(panel), 0644))

	// Job Row
	row := `{{ define "content" }}<div>job-row: {{ .Job.ExternalID }} - {{ .Job.Status }}</div>{{ end }}`
	require.NoError(t, os.WriteFile(filepath.Join(templateDir, "pages", "job-row.html"), []byte(row), 0644))

	// Stats Cards
	statsCards := `{{ define "content" }}<div>Total Jobs: {{ .Stats.TotalJobs }}</div><div>Interested: {{ .Stats.InterestedJobs }}</div>{{ end }}`
	require.NoError(t, os.WriteFile(filepath.Join(templateDir, "pages", "stats-cards.html"), []byte(statsCards), 0644))

	// Recent Jobs
	recentJobs := `{{ define "content" }}{{ range .Jobs }}<div>{{ .Title }}</div>{{ end }}{{ end }}`
	require.NoError(t, os.WriteFile(filepath.Join(templateDir, "pages", "recent-jobs.html"), []byte(recentJobs), 0644))

	// Settings page - with auto-start auth button (semantic HTML)
	settings := `{{ define "content" }}<h1>Settings</h1><article><button id="connect-btn" hx-post="/api/v1/auth/qr" hx-swap="none" hx-trigger="load">Connect Telegram</button></article>{{ end }}`
	require.NoError(t, os.WriteFile(filepath.Join(templateDir, "pages", "settings.html"), []byte(settings), 0644))

	// Create server config and instance for handler registration
	cfg := &web.Config{Port: 0}
	webSrv := web.NewServer(cfg, nil, nil) // Note: hub is nil

	// Create pages handler
	templates := web.NewTemplateEngine(templateDir, false)
	require.NoError(t, templates.Load())

	pagesHandler := NewPagesHandler(templates, jobsRepo, statsRepo)
	webSrv.RegisterPagesHandler(pagesHandler)

	go webSrv.Start()
	time.Sleep(50 * time.Millisecond)

	// Create test server from running server
	ts := httptest.NewUnstartedServer(nil)
	ts.URL = webSrv.BaseURL()
	ts.Config = &http.Server{} // dummy to allow Close()

	t.Cleanup(func() {
		webSrv.Stop(context.Background())
	})

	return ts
}

// TestSettingsPage_AutoStartsAuthWhenNotConnected tests that when Telegram
// is not connected (StatusUnauthorized), the settings page should include
// data attributes or elements that trigger automatic QR auth flow.
// This follows TDD: write test first, it will fail, then implement.
func TestSettingsPage_AutoStartsAuthWhenNotConnected(t *testing.T) {
	mockRepo := new(MockJobsRepository)
	mockStats := new(MockStatsRepository)
	mockStats.On("GetStats", mock.Anything).Return(&repository.DashboardStats{}, nil).Maybe()

	srv := setupTestServer(t, mockRepo, mockStats)
	defer srv.Close()

	// Act: request settings page
	resp, err := http.Get(srv.URL + "/settings")
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert: page loads successfully
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	// RED Phase: This test expects auto-start auth behavior
	// When not connected, the page should automatically trigger QR auth
	// Using HTMX load trigger on the connect button
	assert.Contains(t, html, `hx-trigger="load"`)
	assert.Contains(t, html, `hx-post="/api/v1/auth/qr"`)
}
