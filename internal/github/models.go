package github

import "github.com/bigwhite/issue2md/internal/urlparse"

// Content 是 Issue/PR/Discussion 的统一内容模型，供渲染层消费。
type Content struct {
	Kind      urlparse.Kind
	Title     string
	Body      string
	Author    string
	State     string
	CreatedAt string
	Labels    []string
	Comments  []Comment

	// PR 特有字段（非 PR 时为零值）
	Merged       bool
	Additions    int
	Deletions    int
	ChangedFiles int
}

// Comment 表示一条评论（含 reactions 总数）。
type Comment struct {
	Author    string
	CreatedAt string
	Body      string
	Reactions int
}
