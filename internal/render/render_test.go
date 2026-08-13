package render

import (
	"strings"
	"testing"

	"github.com/bigwhite/issue2md/internal/github"
	"github.com/bigwhite/issue2md/internal/urlparse"
)

func TestRenderIssue(t *testing.T) {
	content := &github.Content{
		Kind:      urlparse.KindIssue,
		Title:     "Bug: crash on startup",
		Body:      "When I run the app it crashes.",
		Author:    "alice",
		State:     "open",
		CreatedAt: "2026-08-01T10:00:00Z",
		Labels:    []string{"bug", "high-priority"},
		Comments: []github.Comment{
			{Author: "bob", CreatedAt: "2026-08-02T10:00:00Z", Body: "Try version 2.0, it works for me.", Reactions: 5},
		},
	}
	target := &urlparse.Target{Kind: urlparse.KindIssue, Owner: "octocat", Repo: "Hello-World", Number: 123}

	got := Render(content, target, content.Comments)

	for _, want := range []string{
		"# Bug: crash on startup",
		"> 状态：open · 作者：alice · 创建时间：2026-08-01T10:00:00Z · labels：bug, high-priority",
		"> 评论数：1 · 原始链接：https://github.com/octocat/Hello-World/issues/123",
		"When I run the app it crashes.",
		"## 💬 精选评论",
		"> 作者：bob · 时间：2026-08-02T10:00:00Z · 点赞：5",
		"Try version 2.0, it works for me.",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Render() output missing %q\n--- output ---\n%s", want, got)
		}
	}
}

func TestRenderPullIncludesMergeStats(t *testing.T) {
	content := &github.Content{
		Kind:         urlparse.KindPull,
		Title:        "Fix crash on startup",
		Body:         "This PR fixes the crash.",
		Author:       "alice",
		State:        "closed",
		CreatedAt:    "2026-08-01T10:00:00Z",
		Merged:       true,
		Additions:    42,
		Deletions:    10,
		ChangedFiles: 3,
	}
	target := &urlparse.Target{Kind: urlparse.KindPull, Owner: "octocat", Repo: "Hello-World", Number: 456}

	got := Render(content, target, nil)

	for _, want := range []string{
		"> 合并：已合并 · 变更文件：3 · 增删行：+42/-10",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Render() output missing %q\n--- output ---\n%s", want, got)
		}
	}
	if strings.Contains(got, "精选评论") {
		t.Errorf("Render() should omit 精选评论 section when no selected comments\n--- output ---\n%s", got)
	}
}

func TestRenderUnmergedPull(t *testing.T) {
	content := &github.Content{
		Kind:      urlparse.KindPull,
		Title:     "Draft PR",
		Author:    "alice",
		State:     "open",
		CreatedAt: "2026-08-01T10:00:00Z",
	}
	target := &urlparse.Target{Kind: urlparse.KindPull, Owner: "octocat", Repo: "Hello-World", Number: 456}

	got := Render(content, target, nil)

	if !strings.Contains(got, "> 合并：未合并") {
		t.Errorf("Render() output missing 未合并\n--- output ---\n%s", got)
	}
}

func TestRenderNoLabels(t *testing.T) {
	content := &github.Content{
		Kind:      urlparse.KindIssue,
		Title:     "No labels",
		Author:    "alice",
		State:     "open",
		CreatedAt: "2026-08-01T10:00:00Z",
	}
	target := &urlparse.Target{Kind: urlparse.KindIssue, Owner: "octocat", Repo: "Hello-World", Number: 123}

	got := Render(content, target, nil)

	if !strings.Contains(got, "labels：-") {
		t.Errorf("Render() output missing labels：-\n--- output ---\n%s", got)
	}
}

func TestRenderMultipleSelectedComments(t *testing.T) {
	content := &github.Content{
		Kind:      urlparse.KindIssue,
		Title:     "Title",
		Author:    "alice",
		State:     "open",
		CreatedAt: "2026-08-01T10:00:00Z",
	}
	target := &urlparse.Target{Kind: urlparse.KindIssue, Owner: "octocat", Repo: "Hello-World", Number: 123}
	selected := []github.Comment{
		{Author: "bob", CreatedAt: "2026-08-02T10:00:00Z", Body: "First, solved.", Reactions: 9},
		{Author: "carol", CreatedAt: "2026-08-03T10:00:00Z", Body: "Second, it works.", Reactions: 3},
	}

	got := Render(content, target, selected)

	if !strings.Contains(got, "First, solved.") || !strings.Contains(got, "Second, it works.") {
		t.Errorf("Render() should include all selected comments\n--- output ---\n%s", got)
	}
	if strings.Index(got, "First, solved.") > strings.Index(got, "Second, it works.") {
		t.Errorf("Render() comments order differs from selected order\n--- output ---\n%s", got)
	}
}
