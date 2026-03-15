package api

import (
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// ConflictResponse represents a detected conflict between contexts.
type ConflictResponse struct {
	ID              int        `json:"id"`
	ContextATopic   string     `json:"context_a_topic"`
	ContextBTopic   string     `json:"context_b_topic"`
	ContextAVersion string     `json:"context_a_version,omitempty"`
	ContextBVersion string     `json:"context_b_version,omitempty"`
	ConflictType    string     `json:"conflict_type"`
	Severity        string     `json:"severity"`
	Description     string     `json:"description"`
	Status          string     `json:"status"`
	Resolution      *string    `json:"resolution,omitempty"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// ConflictListResponse wraps a list of conflicts.
type ConflictListResponse struct {
	Conflicts []ConflictResponse `json:"conflicts"`
	Total     int                `json:"total"`
}

// ConflictDetectRequest triggers conflict detection.
type ConflictDetectRequest struct {
	Scope string `json:"scope,omitempty"` // "all" or "recent"
}

// ConflictDetectResponse is the result of conflict detection.
type ConflictDetectResponse struct {
	ConflictsFound int `json:"conflicts_found"`
	Scanned        int `json:"scanned"`
}

// ConflictResolveRequest resolves a conflict.
type ConflictResolveRequest struct {
	Resolution string `json:"resolution"` // "dismiss", "keep_a", "keep_b", "merge"
}

// ListConflicts fetches conflicts with optional status filter.
func (c *Client) ListConflicts(project, status string, limit, offset int) (*ConflictListResponse, error) {
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

	var resp ConflictListResponse
	if err := c.Get(fmt.Sprintf("/projects/%s/conflicts", project), params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetConflict fetches a single conflict by ID.
func (c *Client) GetConflict(project string, id int) (*ConflictResponse, error) {
	var resp ConflictResponse
	if err := c.Get(fmt.Sprintf("/projects/%s/conflicts/%d", project, id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DetectConflicts triggers conflict detection.
func (c *Client) DetectConflicts(project string, req ConflictDetectRequest) (*ConflictDetectResponse, error) {
	var resp ConflictDetectResponse
	if err := c.Post(fmt.Sprintf("/projects/%s/conflicts/detect", project), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ResolveConflict resolves a conflict by ID.
func (c *Client) ResolveConflict(project string, id int, req ConflictResolveRequest) (*ConflictResponse, error) {
	var resp ConflictResponse
	if err := c.Post(fmt.Sprintf("/projects/%s/conflicts/%d/resolve", project, id), req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
