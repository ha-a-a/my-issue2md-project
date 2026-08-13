// Package render 将 GitHub 内容渲染为规范的 Markdown 文档。
package render

import (
	"strconv"
	"strings"

	"github.com/bigwhite/issue2md/internal/github"
	"github.com/bigwhite/issue2md/internal/urlparse"
)

// Render 生成 Markdown 文档：
//   - 元数据头部（标题、状态、作者、时间、labels、评论数、原始链接）
//   - 正文（PR 额外包含合并状态与增删行统计）
//   - 精选评论章节（selected 为空时省略）
func Render(content *github.Content, target *urlparse.Target, selected []github.Comment) string {
	var b strings.Builder

	b.WriteString("# ")
	b.WriteString(content.Title)
	b.WriteString("\n\n")

	b.WriteString("> 状态：")
	b.WriteString(content.State)
	b.WriteString(" · 作者：")
	b.WriteString(content.Author)
	b.WriteString(" · 创建时间：")
	b.WriteString(content.CreatedAt)
	b.WriteString(" · labels：")
	b.WriteString(labelsString(content.Labels))
	b.WriteString("\n")

	if content.Kind == urlparse.KindPull {
		merged := "未合并"
		if content.Merged {
			merged = "已合并"
		}
		b.WriteString("> 合并：")
		b.WriteString(merged)
		b.WriteString(" · 变更文件：")
		b.WriteString(strconv.Itoa(content.ChangedFiles))
		b.WriteString(" · 增删行：+")
		b.WriteString(strconv.Itoa(content.Additions))
		b.WriteString("/-")
		b.WriteString(strconv.Itoa(content.Deletions))
		b.WriteString("\n")
	}

	b.WriteString("> 评论数：")
	b.WriteString(strconv.Itoa(len(content.Comments)))
	b.WriteString(" · 原始链接：")
	b.WriteString(target.URL())
	b.WriteString("\n\n")

	if content.Body != "" {
		b.WriteString(content.Body)
		b.WriteString("\n")
	}

	if len(selected) > 0 {
		b.WriteString("\n## 💬 精选评论\n\n")
		for i, cm := range selected {
			b.WriteString("> 作者：")
			b.WriteString(cm.Author)
			b.WriteString(" · 时间：")
			b.WriteString(cm.CreatedAt)
			b.WriteString(" · 点赞：")
			b.WriteString(strconv.Itoa(cm.Reactions))
			b.WriteString("\n\n")
			b.WriteString(cm.Body)
			b.WriteString("\n")
			if i < len(selected)-1 {
				b.WriteString("\n---\n\n")
			}
		}
	}

	return b.String()
}

// labelsString 渲染 labels 列表，为空时显示 "-"。
func labelsString(labels []string) string {
	if len(labels) == 0 {
		return "-"
	}
	return strings.Join(labels, ", ")
}
