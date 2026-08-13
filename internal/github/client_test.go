package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bigwhite/issue2md/internal/urlparse"
)

func TestFetchIssue(t *testing.T) {
	srv := newTestServer(t, map[string]jsonResponse{
		"/repos/octocat/Hello-World/issues/123": {
			body: `{
				"title": "Bug: crash on startup",
				"body": "When I run the app it crashes.",
				"state": "open",
				"created_at": "2026-08-01T10:00:00Z",
				"user": {"login": "alice"},
				"labels": [{"name": "bug"}, {"name": "high-priority"}],
				"comments": 2
			}`,
		},
		"/repos/octocat/Hello-World/issues/123/comments": {
			body: `[
				{"user": {"login": "bob"}, "created_at": "2026-08-02T10:00:00Z", "body": "Try version 2.0, it works for me.", "reactions": {"total_count": 5}},
				{"user": {"login": "carol"}, "created_at": "2026-08-03T10:00:00Z", "body": "Upgrade Go to 1.24 fixed it.", "reactions": {"total_count": 3}}
			]`,
		},
	})

	client := NewClient("")
	target := &urlparse.Target{Kind: urlparse.KindIssue, Owner: "octocat", Repo: "Hello-World", Number: 123}
	client.baseURL = srv.URL

	content, err := client.Fetch(target)
	if err != nil {
		t.Fatalf("Fetch() unexpected error: %v", err)
	}

	if content.Title != "Bug: crash on startup" {
		t.Errorf("Title = %q, want %q", content.Title, "Bug: crash on startup")
	}
	if content.Author != "alice" {
		t.Errorf("Author = %q, want %q", content.Author, "alice")
	}
	if content.State != "open" {
		t.Errorf("State = %q, want %q", content.State, "open")
	}
	if len(content.Labels) != 2 || content.Labels[0] != "bug" || content.Labels[1] != "high-priority" {
		t.Errorf("Labels = %v, want [bug high-priority]", content.Labels)
	}
	if len(content.Comments) != 2 {
		t.Fatalf("len(Comments) = %d, want 2", len(content.Comments))
	}
	if content.Comments[0].Reactions != 5 {
		t.Errorf("Comments[0].Reactions = %d, want 5", content.Comments[0].Reactions)
	}
}

func TestFetchPull(t *testing.T) {
	srv := newTestServer(t, map[string]jsonResponse{
		"/repos/octocat/Hello-World/pulls/456": {
			body: `{
				"title": "Fix crash on startup",
				"body": "This PR fixes the crash.",
				"state": "closed",
				"created_at": "2026-08-01T10:00:00Z",
				"user": {"login": "alice"},
				"labels": [],
				"comments": 0,
				"merged": true,
				"additions": 42,
				"deletions": 10,
				"changed_files": 3
			}`,
		},
		"/repos/octocat/Hello-World/issues/456/comments": {body: "[]"},
	})

	client := NewClient("")
	target := &urlparse.Target{Kind: urlparse.KindPull, Owner: "octocat", Repo: "Hello-World", Number: 456}
	client.baseURL = srv.URL

	content, err := client.Fetch(target)
	if err != nil {
		t.Fatalf("Fetch() unexpected error: %v", err)
	}

	if !content.Merged {
		t.Errorf("Merged = false, want true")
	}
	if content.Additions != 42 {
		t.Errorf("Additions = %d, want 42", content.Additions)
	}
	if content.Deletions != 10 {
		t.Errorf("Deletions = %d, want 10", content.Deletions)
	}
	if content.ChangedFiles != 3 {
		t.Errorf("ChangedFiles = %d, want 3", content.ChangedFiles)
	}
}

func TestFetchNotFound(t *testing.T) {
	srv := newTestServer(t, map[string]jsonResponse{
		"/repos/octocat/Hello-World/issues/999": {
			status: http.StatusNotFound,
			body:   `{"message": "Not Found"}`,
		},
	})

	client := NewClient("")
	target := &urlparse.Target{Kind: urlparse.KindIssue, Owner: "octocat", Repo: "Hello-World", Number: 999}
	client.baseURL = srv.URL

	_, err := client.Fetch(target)
	if err == nil {
		t.Fatal("Fetch() expected error for 404, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error = %q, want it to mention status 404", err.Error())
	}
}

func TestFetchRateLimited(t *testing.T) {
	srv := newTestServer(t, map[string]jsonResponse{
		"/repos/octocat/Hello-World/issues/123": {
			status: http.StatusForbidden,
			body:   `{"message": "API rate limit exceeded"}`,
		},
	})

	client := NewClient("")
	target := &urlparse.Target{Kind: urlparse.KindIssue, Owner: "octocat", Repo: "Hello-World", Number: 123}
	client.baseURL = srv.URL

	_, err := client.Fetch(target)
	if err == nil {
		t.Fatal("Fetch() expected error for 403, got nil")
	}
}

func TestFetchDiscussion(t *testing.T) {
	var gotAuthHeader string
	var gotGraphQLBody graphQLRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuthHeader = r.Header.Get("Authorization")
		if r.Method != http.MethodPost || r.URL.Path != "/graphql" {
			http.Error(w, fmt.Sprintf("unexpected request: %s %s", r.Method, r.URL.Path), http.StatusInternalServerError)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&gotGraphQLBody); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"repository": {
					"discussion": {
						"title": "How to use issue2md?",
						"body": "I want to convert issues to markdown.",
						"createdAt": "2026-08-01T10:00:00Z",
						"closed": true,
						"author": {"login": "alice"},
						"labels": {"nodes": [{"name": "question"}]},
						"comments": {
							"totalCount": 2,
							"nodes": [
								{"author": {"login": "bob"}, "createdAt": "2026-08-02T10:00:00Z", "body": "It works!", "reactions": {"totalCount": 7}},
								{"author": {"login": "carol"}, "createdAt": "2026-08-03T10:00:00Z", "body": "Solved my problem.", "reactions": {"totalCount": 2}}
							]
						}
					}
				}
			}
		}`))
	}))
	defer srv.Close()

	client := NewClient("test-token-123")
	target := &urlparse.Target{Kind: urlparse.KindDiscussion, Owner: "octocat", Repo: "Hello-World", Number: 789}
	client.baseURL = srv.URL

	content, err := client.Fetch(target)
	if err != nil {
		t.Fatalf("Fetch() unexpected error: %v", err)
	}

	if gotAuthHeader != "Bearer test-token-123" {
		t.Errorf("Authorization header = %q, want %q", gotAuthHeader, "Bearer test-token-123")
	}
	if gotGraphQLBody.Query == "" {
		t.Error("GraphQL query is empty")
	}
	if gotGraphQLBody.Variables.Number != 789 {
		t.Errorf("GraphQL number = %d, want 789", gotGraphQLBody.Variables.Number)
	}
	if content.Title != "How to use issue2md?" {
		t.Errorf("Title = %q, want %q", content.Title, "How to use issue2md?")
	}
	if content.State != "closed" {
		t.Errorf("State = %q, want %q", content.State, "closed")
	}
	if len(content.Comments) != 2 {
		t.Fatalf("len(Comments) = %d, want 2", len(content.Comments))
	}
	if content.Comments[0].Reactions != 7 {
		t.Errorf("Comments[0].Reactions = %d, want 7", content.Comments[0].Reactions)
	}
}

func TestFetchDiscussionRequiresToken(t *testing.T) {
	client := NewClient("")
	target := &urlparse.Target{Kind: urlparse.KindDiscussion, Owner: "octocat", Repo: "Hello-World", Number: 789}

	_, err := client.Fetch(target)
	if err == nil {
		t.Fatal("Fetch() expected error for discussion without token, got nil")
	}
	if !strings.Contains(err.Error(), "GITHUB_TOKEN") {
		t.Errorf("error = %q, want it to mention GITHUB_TOKEN", err.Error())
	}
}

func TestFetchDiscussionGraphQLError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": null,
			"errors": [{"message": "Could not resolve to a Discussion with the number of 789."}]
		}`))
	}))
	defer srv.Close()

	client := NewClient("test-token")
	target := &urlparse.Target{Kind: urlparse.KindDiscussion, Owner: "octocat", Repo: "Hello-World", Number: 789}
	client.baseURL = srv.URL

	_, err := client.Fetch(target)
	if err == nil {
		t.Fatal("Fetch() expected error for GraphQL errors, got nil")
	}
}

// ---- test helpers ----

type jsonResponse struct {
	status int
	body   string
}

func newTestServer(t *testing.T, routes map[string]jsonResponse) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp, ok := routes[r.URL.Path]
		if !ok {
			http.Error(w, fmt.Sprintf("unexpected path: %s", r.URL.Path), http.StatusInternalServerError)
			return
		}
		status := resp.status
		if status == 0 {
			status = http.StatusOK
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(resp.body))
	}))
	t.Cleanup(srv.Close)
	return srv
}
