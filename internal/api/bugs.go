package api

import (
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// BugReportResponse represents a bug report.
type BugReportResponse struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Severity    string    `json:"severity,omitempty"`
	Steps       string    `json:"steps_to_reproduce,omitempty"`
	Expected    string    `json:"expected_behavior,omitempty"`
	Actual      string    `json:"actual_behavior,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
}

// BugReportListResponse wraps a list of bug reports.
type BugReportListResponse struct {
	BugReports []BugReportResponse `json:"bug_reports"`
	Total      int                 `json:"total"`
}

// ListBugReports fetches bug reports with optional status filter.
func (c *Client) ListBugReports(project, status string, limit, offset int) (*BugReportListResponse, error) {
	params := url.Values{}
	if status != "" {
		params.Set("status", status)
	}
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		params.Set("offset", strconv.Itoa(offset))
	}

	var resp BugReportListResponse
	if err := c.Get(fmt.Sprintf("/projects/%s/bug-reports", project), params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetBugReport fetches a single bug report by ID.
func (c *Client) GetBugReport(project string, id int) (*BugReportResponse, error) {
	var resp BugReportResponse
	if err := c.Get(fmt.Sprintf("/projects/%s/bug-reports/%d", project, id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
