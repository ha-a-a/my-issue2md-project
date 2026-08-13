# issue2md 需求规格说明书
# Version: 0.1.0, 2026-08-13

## 1. 概述

`issue2md` 是一个命令行工具：输入一个 GitHub **Issue / PR / Discussion** 的 URL，自动拉取其内容并转换为规范的 Markdown 文档，保存为本地文件。

**核心价值**：不仅导出正文，更从评论中挖掘**高价值评论**（被社区验证过 "it works" 的解决方案），生成可归档、可分享的 Markdown 文档。

## 2. 输入与输出

- **输入**：一个或多个 GitHub URL（命令行参数），格式：
  - `https://github.com/{owner}/{repo}/issues/{number}` — Issue
  - `https://github.com/{owner}/{repo}/pull/{number}` — PR
  - `https://github.com/{owner}/{repo}/discussions/{number}` — Discussion
  - URL 可带查询参数或 fragment（如 `?foo=bar`、`#issuecomment-123`），应被忽略。
- **输出**：每个 URL 生成一个 `.md` 文件保存到当前目录（或 `-o <dir>` 指定目录）。
- **文件命名**：`{repo}_{kind}_{number}.md`，其中 `kind` ∈ `issue` | `pull` | `discussion`。

## 3. 内容结构

生成的 Markdown 文档包含三部分：

### 3.1 元数据头部（完整元数据）

```
# {title}

> 状态：{open|closed} · 作者：{author} · 创建时间：{created_at} · labels：{labels}
> 评论数：{comments_count} · 原始链接：{url}
```

- 标题、状态、作者、创建时间、labels、评论总数、原始 URL。

### 3.2 正文

- Issue/PR 的 body（原始 Markdown，原样保留）。
- PR 额外包含：合并状态、变更文件数、增删行数（`additions`/`deletions`）。
- Discussion 的 body 及作者信息。

### 3.3 高价值评论（精选评论，最多 3 条）

按以下算法从全部评论中筛选：

1. **it works 双确认**：评论正文包含内置中英文关键词 **且** reactions 总数 > 0。
   - 内置关键词（不区分大小写）：
     - 英文：`it works`、`worked`、`worked for me`、`solved`、`fixed`
     - 中文：`解决了`、`有效`、`可以了`、`成功了`、`搞定了`
2. **排序**：在 it works 评论中按 **reactions 总和**（👍❤️🎉👀 等所有 reactions 之和）降序排列。
3. **选取**：取前 3 条。
4. **兜底**：若没有 it works 评论，则取 reactions 总和最高的前 3 条普通评论。
5. **边界**：有效评论不足 3 条时，有多少输出多少；一条都没有则省略该章节。

每条精选评论的 Markdown 格式：

```
## 💬 精选评论

> 作者：{author} · 时间：{created_at} · 点赞：{reactions_total}

{comment_body}
```

## 4. 认证

- **可选 `GITHUB_TOKEN`**：优先读取环境变量 `GITHUB_TOKEN`，设置后所有 API 请求携带 `Authorization: Bearer <token>`（限 5000 次/小时）。
- **匿名回退**：未设置 token 时以匿名访问（限 60 次/小时）。
- **Discussion 特殊约束**：GitHub REST API 不提供 Discussion 内容端点，必须通过 **GraphQL API** 获取，而 GraphQL 需要认证。因此：
  - 匿名访问 Discussion URL 时，报错退出并提示设置 `GITHUB_TOKEN`。

## 5. CLI 接口

```
issue2md [flags] <url> [<url>...]

Flags:
  -o <dir>      输出目录（默认当前目录）
  -h            显示帮助
```

- 支持一次传入多个 URL 批处理，逐个转换。
- 单个 URL 处理失败时打印错误并继续处理其余 URL（不整体中断）；退出码非 0 表示至少有一个失败。

## 6. 技术约束

- Go >= 1.24，标准库优先（HTTP 使用 `net/http`，CLI 使用 `flag` 包），不引入第三方依赖。
- 测试先行（TDD，Red-Green-Refactor），单元测试采用表格驱动测试（Table-Driven Tests）。
- 遵循项目宪法：简单性原则、明确性原则（错误显式处理、`fmt.Errorf("...: %w", err)`、无全局变量）。

## 7. 非目标 (Non-goals)

- 不下载/嵌入正文中的图片资源（保留原始 URL 引用）。
- 不转换 Markdown 之外的格式（不输出 HTML、PDF）。
- 不将 Markdown 写回 GitHub（只读导出）。
- 不支持 GitHub Enterprise 域名（仅 `github.com`）。
- 不处理 Discussion 的嵌套回复（GraphQL 评论仅取第一层）。
