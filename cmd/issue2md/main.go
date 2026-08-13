// issue2md 是一个命令行工具：输入 GitHub Issue/PR/Discussion 的 URL，
// 自动将其转换为规范的 Markdown 文档保存到本地文件。
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bigwhite/issue2md/internal/commentfilter"
	"github.com/bigwhite/issue2md/internal/github"
	"github.com/bigwhite/issue2md/internal/render"
	"github.com/bigwhite/issue2md/internal/urlparse"
)

func main() {
	outDir := flag.String("o", ".", "输出目录（默认当前目录）")
	flag.Usage = usage
	flag.Parse()

	urls := flag.Args()
	if len(urls) == 0 {
		usage()
		os.Exit(2)
	}

	client := github.NewClient(os.Getenv("GITHUB_TOKEN"))

	failures := 0
	for _, rawURL := range urls {
		if err := convert(rawURL, *outDir, client); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			failures++
		}
	}

	if failures > 0 {
		fmt.Fprintf(os.Stderr, "%d of %d URL(s) failed\n", failures, len(urls))
		os.Exit(1)
	}
}

// convert 转换单个 URL：解析 → 拉取 → 筛选评论 → 渲染 → 写文件。
func convert(rawURL, outDir string, client *github.Client) error {
	target, err := urlparse.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse url %q: %w", rawURL, err)
	}

	content, err := client.Fetch(target)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", target.URL(), err)
	}

	selected := commentfilter.Select(content.Comments)
	md := render.Render(content, target, selected)

	path := filepath.Join(outDir, fmt.Sprintf("%s_%s_%d.md", target.Repo, target.Kind, target.Number))
	if err := os.WriteFile(path, []byte(md), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	fmt.Printf("✓ %s -> %s\n", target.URL(), path)
	return nil
}

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), `issue2md - 将 GitHub Issue/PR/Discussion 转换为 Markdown 文档

用法：
  issue2md [-o 输出目录] <url> [<url>...]

参数：
  <url>  GitHub Issue/PR/Discussion URL，如 https://github.com/owner/repo/issues/123
  -o     输出目录（默认当前目录）
  -h     显示帮助

环境变量：
  GITHUB_TOKEN  可选，设置后使用认证访问 API（限 5000 次/小时）；
                未设置时匿名访问（限 60 次/小时）。转换 Discussion 时必须设置。
`)
}
