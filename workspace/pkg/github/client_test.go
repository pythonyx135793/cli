package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetIssues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		issues := []Issue{
			{ID: 1, Title: "Issue 1"},
			{ID: 2, Title: "Issue 2"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(issues)
	}))
	defer server.Close()

	client := NewClient(server.URL, nil)
	issues, err := client.GetIssues("owner", "repo", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(issues) != 2 {
		t.Errorf("expected 2 issues, got %d", len(issues))
	}
}

func TestGetIssues_SparsePagination(t *testing.T) {
	requestCount := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		if requestCount == 1 {
			issues := []Issue{
				{ID: 1, Title: "Issue 1"},
				{ID: 2, Title: "Issue 2"},
			}
			w.Header().Set("Link", `<`+server.URL+`/repos/owner/repo/issues?page=2&per_page=5>; rel="next"`)
			json.NewEncoder(w).Encode(issues)
		} else if requestCount == 2 {
			issues := []Issue{
				{ID: 3, Title: "Issue 3"},
				{ID: 4, Title: "Issue 4"},
				{ID: 5, Title: "Issue 5"},
			}
			json.NewEncoder(w).Encode(issues)
		} else {
			t.Errorf("unexpected request count: %d", requestCount)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, nil)
	issues, err := client.GetIssues("owner", "repo", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if requestCount != 2 {
		t.Errorf("expected exactly 2 requests, got %d", requestCount)
	}

	if len(issues) != 5 {
		t.Errorf("expected 5 issues, got %d", len(issues))
	}
}

func TestGetIssues_EmptyPageStopsEvenWithNextLink(t *testing.T) {
	requestCount := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		if requestCount == 1 {
			w.Header().Set("Link", `<`+server.URL+`/repos/owner/repo/issues?page=2&per_page=3>; rel="next"`)
			json.NewEncoder(w).Encode([]Issue{})
		} else if requestCount == 2 {
			issues := []Issue{
				{ID: 1, Title: "Issue 1"},
				{ID: 2, Title: "Issue 2"},
				{ID: 3, Title: "Issue 3"},
				{ID: 4, Title: "Issue 4"},
			}
			json.NewEncoder(w).Encode(issues)
		} else {
			t.Errorf("unexpected request count: %d", requestCount)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, nil)
	issues, err := client.GetIssues("owner", "repo", 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if requestCount != 1 {
		t.Errorf("expected exactly 1 request, got %d", requestCount)
	}

	if len(issues) != 0 {
		t.Errorf("expected no issues after an empty first page, got %d", len(issues))
	}
}

func TestGetIssues_FifteenThenTenWithThirtyRequested(t *testing.T) {
	requestCount := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("per_page") != "30" {
			t.Errorf("expected per_page=30, got %q", r.URL.RawQuery)
		}
		first, last := 1, 15
		if r.URL.Query().Get("page") == "2" {
			first, last = 16, 25
		} else {
			w.Header().Set("Link", `<`+server.URL+`/repos/owner/repo/issues?page=2&per_page=30>; rel="next"`)
		}
		var page []Issue
		for id := first; id <= last; id++ {
			page = append(page, Issue{ID: id, Title: fmt.Sprintf("Issue %d", id)})
		}
		json.NewEncoder(w).Encode(page)
	}))
	defer server.Close()

	issues, err := NewClient(server.URL, nil).GetIssues("owner", "repo", 30)
	if err != nil {
		t.Fatal(err)
	}
	if requestCount != 2 || len(issues) != 25 {
		t.Fatalf("expected 2 requests and 25 issues, got %d requests and %d issues", requestCount, len(issues))
	}
	for i, issue := range issues {
		if issue.ID != i+1 {
			t.Errorf("issue %d: expected ID %d, got %d", i, i+1, issue.ID)
		}
	}
}

func TestGetIssues_StopsAtLimitEvenWithNextLink(t *testing.T) {
	requestCount := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Link", fmt.Sprintf(`<%s/repos/owner/repo/issues?page=%d&per_page=3>; rel="next"`, server.URL, requestCount+1))
		if requestCount > 2 {
			t.Errorf("requested a page after reaching the limit")
		}
		first := requestCount*2 - 1
		json.NewEncoder(w).Encode([]Issue{{ID: first}, {ID: first + 1}})
	}))
	defer server.Close()

	issues, err := NewClient(server.URL, nil).GetIssues("owner", "repo", 3)
	if err != nil {
		t.Fatal(err)
	}
	if requestCount != 2 || len(issues) != 3 {
		t.Fatalf("expected 2 requests and 3 issues, got %d requests and %d issues", requestCount, len(issues))
	}
	for i, issue := range issues {
		if issue.ID != i+1 {
			t.Errorf("issue %d: expected ID %d, got %d", i, i+1, issue.ID)
		}
	}
}

func TestGetIssues_StopsWithoutNextLinkBeforeLimit(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Issue{{ID: 1}, {ID: 2}})
	}))
	defer server.Close()

	issues, err := NewClient(server.URL, nil).GetIssues("owner", "repo", 5)
	if err != nil {
		t.Fatal(err)
	}
	if requestCount != 1 || len(issues) != 2 {
		t.Fatalf("expected 1 request and 2 issues, got %d requests and %d issues", requestCount, len(issues))
	}
}
