package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetenvFallback(t *testing.T) {
	t.Setenv("TENANT_TEST_VALUE", "")
	if got := getenv("TENANT_TEST_VALUE", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}
}

func TestGetenvValue(t *testing.T) {
	t.Setenv("TENANT_TEST_VALUE", "tenant-a")
	if got := getenv("TENANT_TEST_VALUE", "fallback"); got != "tenant-a" {
		t.Fatalf("expected env value, got %q", got)
	}
}

func TestHealthz_DBUnavailable(t *testing.T) {
	s := &server{tenant: "tenant-a", db: nil}

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	s.healthz(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}

	var res map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res["error"] == "" {
		t.Fatal("expected error message in response body")
	}
}

func TestCreateItem_Validation(t *testing.T) {
	s := &server{tenant: "tenant-a"}

	tests := []struct {
		name         string
		body         string
		expectedCode int
		expectedErr  string
	}{
		{
			name:         "empty json",
			body:         `{}`,
			expectedCode: http.StatusBadRequest,
			expectedErr:  "name is required",
		},
		{
			name:         "empty string name",
			body:         `{"name":""}`,
			expectedCode: http.StatusBadRequest,
			expectedErr:  "name is required",
		},
		{
			name:         "whitespace-only name",
			body:         `{"name":"   "}`,
			expectedCode: http.StatusBadRequest,
			expectedErr:  "name is required",
		},
		{
			name:         "invalid json body",
			body:         `{bad-json}`,
			expectedCode: http.StatusBadRequest,
			expectedErr:  "invalid character",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/items", bytes.NewBufferString(tc.body))
			rec := httptest.NewRecorder()

			s.createItem(rec, req)

			if rec.Code != tc.expectedCode {
				t.Fatalf("expected status %d, got %d", tc.expectedCode, rec.Code)
			}
		})
	}
}

func TestVote_Validation(t *testing.T) {
	s := &server{tenant: "tenant-a"}

	tests := []struct {
		name         string
		body         string
		expectedCode int
	}{
		{
			name:         "invalid json",
			body:         `{invalid}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "missing voter_id",
			body:         `{"score": 5, "comment": "great"}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "whitespace voter_id",
			body:         `{"voter_id": "   ", "score": 5}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "score too low (0)",
			body:         `{"voter_id": "user-1", "score": 0}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "score too high (6)",
			body:         `{"voter_id": "user-1", "score": 6}`,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "negative score",
			body:         `{"voter_id": "user-1", "score": -2}`,
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/items/1/vote", bytes.NewBufferString(tc.body))
			rec := httptest.NewRecorder()

			s.vote(rec, req)

			if rec.Code != tc.expectedCode {
				t.Fatalf("expected status %d, got %d", tc.expectedCode, rec.Code)
			}
		})
	}
}

func TestExport_NotConfigured(t *testing.T) {
	s := &server{tenant: "tenant-a", exportBucket: "", s3: nil}

	req := httptest.NewRequest(http.MethodPost, "/exports", nil)
	rec := httptest.NewRecorder()

	s.export(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rec.Code)
	}

	var res map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res["error"] != "S3 export is not configured" {
		t.Fatalf("expected 'S3 export is not configured', got %q", res["error"])
	}
}

func TestWriteJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	payload := map[string]string{"message": "hello"}

	writeJSON(rec, http.StatusOK, payload)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}

	var res map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if res["message"] != "hello" {
		t.Fatalf("expected message 'hello', got %q", res["message"])
	}
}

func TestWriteError(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, http.StatusNotFound, errors.New("resource not found"))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", ct)
	}

	var res map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if res["error"] != "resource not found" {
		t.Fatalf("expected 'resource not found', got %q", res["error"])
	}
}
