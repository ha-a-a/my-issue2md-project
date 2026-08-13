package urlparse

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		rawURL  string
		want    *Target
		wantErr bool
	}{
		{
			name:   "issue URL",
			rawURL: "https://github.com/octocat/Hello-World/issues/123",
			want:   &Target{Kind: KindIssue, Owner: "octocat", Repo: "Hello-World", Number: 123},
		},
		{
			name:   "pull URL",
			rawURL: "https://github.com/octocat/Hello-World/pull/456",
			want:   &Target{Kind: KindPull, Owner: "octocat", Repo: "Hello-World", Number: 456},
		},
		{
			name:   "discussion URL",
			rawURL: "https://github.com/octocat/Hello-World/discussions/789",
			want:   &Target{Kind: KindDiscussion, Owner: "octocat", Repo: "Hello-World", Number: 789},
		},
		{
			name:   "URL with query parameters",
			rawURL: "https://github.com/octocat/Hello-World/issues/123?label=bug&assignee=me",
			want:   &Target{Kind: KindIssue, Owner: "octocat", Repo: "Hello-World", Number: 123},
		},
		{
			name:   "URL with fragment",
			rawURL: "https://github.com/octocat/Hello-World/issues/123#issuecomment-4567890",
			want:   &Target{Kind: KindIssue, Owner: "octocat", Repo: "Hello-World", Number: 123},
		},
		{
			name:    "not github.com host",
			rawURL:  "https://gitlab.com/octocat/Hello-World/issues/123",
			wantErr: true,
		},
		{
			name:    "enterprise github host",
			rawURL:  "https://github.example.com/octocat/Hello-World/issues/123",
			wantErr: true,
		},
		{
			name:    "unknown path segment",
			rawURL:  "https://github.com/octocat/Hello-World/labels/bug",
			wantErr: true,
		},
		{
			name:    "non-numeric issue number",
			rawURL:  "https://github.com/octocat/Hello-World/issues/abc",
			wantErr: true,
		},
		{
			name:    "zero issue number",
			rawURL:  "https://github.com/octocat/Hello-World/issues/0",
			wantErr: true,
		},
		{
			name:    "negative issue number",
			rawURL:  "https://github.com/octocat/Hello-World/issues/-5",
			wantErr: true,
		},
		{
			name:    "missing repo",
			rawURL:  "https://github.com/octocat/issues/123",
			wantErr: true,
		},
		{
			name:    "empty URL",
			rawURL:  "",
			wantErr: true,
		},
		{
			name:    "not a URL",
			rawURL:  "not a url",
			wantErr: true,
		},
		{
			name:    "missing scheme",
			rawURL:  "github.com/octocat/Hello-World/issues/123",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.rawURL)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q) expected error, got nil (target=%+v)", tt.rawURL, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.rawURL, err)
			}
			if got == nil || *got != *tt.want {
				t.Errorf("Parse(%q) = %+v, want %+v", tt.rawURL, got, tt.want)
			}
		})
	}
}
