// Package urlparse 解析 GitHub Issue/PR/Discussion 的 URL。
package urlparse

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Kind 表示 GitHub 内容类型。
type Kind int

const (
	KindIssue Kind = iota
	KindPull
	KindDiscussion
)

// String 返回 Kind 的字符串表示，用于输出文件名。
func (k Kind) String() string {
	switch k {
	case KindIssue:
		return "issue"
	case KindPull:
		return "pull"
	case KindDiscussion:
		return "discussion"
	default:
		return "unknown"
	}
}

// pathSegment 返回 Kind 对应的 URL 路径段。
func (k Kind) pathSegment() string {
	switch k {
	case KindIssue:
		return "issues"
	case KindPull:
		return "pull"
	case KindDiscussion:
		return "discussions"
	default:
		return ""
	}
}

// Target 表示一个已解析的 GitHub 内容目标。
type Target struct {
	Kind   Kind
	Owner  string
	Repo   string
	Number int
}

// URL 返回该目标的规范 URL（不含查询参数与 fragment）。
func (t *Target) URL() string {
	return fmt.Sprintf("https://github.com/%s/%s/%s/%d", t.Owner, t.Repo, t.Kind.pathSegment(), t.Number)
}

// Parse 解析 GitHub URL，如 https://github.com/{owner}/{repo}/issues/{number}。
// 支持 issues/pull/discussions 三种路径，忽略查询参数与 fragment。
func Parse(rawURL string) (*Target, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse url %q: %w", rawURL, err)
	}
	if u.Host != "github.com" {
		return nil, fmt.Errorf("unsupported host %q: only github.com is supported", u.Host)
	}

	segs := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(segs) != 4 {
		return nil, fmt.Errorf("invalid github url %q: expected path /{owner}/{repo}/{kind}/{number}", rawURL)
	}
	owner, repo, kindStr, numStr := segs[0], segs[1], segs[2], segs[3]
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("invalid github url %q: owner and repo must not be empty", rawURL)
	}

	var kind Kind
	switch kindStr {
	case "issues":
		kind = KindIssue
	case "pull":
		kind = KindPull
	case "discussions":
		kind = KindDiscussion
	default:
		return nil, fmt.Errorf("unsupported path segment %q: must be issues, pull or discussions", kindStr)
	}

	number, err := strconv.Atoi(numStr)
	if err != nil {
		return nil, fmt.Errorf("invalid number %q: %w", numStr, err)
	}
	if number <= 0 {
		return nil, fmt.Errorf("invalid number %d: must be positive", number)
	}

	return &Target{Kind: kind, Owner: owner, Repo: repo, Number: number}, nil
}
