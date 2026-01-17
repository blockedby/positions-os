package repository

import (
	"testing"
)

func TestDashboardStats_HasWorkflowFields(t *testing.T) {
	// Verify DashboardStats has the new workflow fields
	stats := DashboardStats{}

	// These fields must exist
	_ = stats.TailoredJobs
	_ = stats.TailoredApprovedJobs
	_ = stats.SentJobs
	_ = stats.RespondedJobs

	// Compile-time check - if fields don't exist, this won't compile
	t.Log("DashboardStats has all required workflow fields")
}
