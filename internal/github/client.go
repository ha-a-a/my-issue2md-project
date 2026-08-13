// Package github 提供 GitHub REST/GraphQL API 客户端。
package github

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/bigwhite/issue2md/internal/urlparse"
)

const defaultBaseURL = "https://api.github.com"

// Client 是 GitHub API 客户端。token 为空时以匿名身份访问 REST API。
type Client struct {
	http    *http.Client
	token   string
	baseURL string
}

// NewClient 创建 GitHub API 客户端。token 为空表示匿名访问。
func NewClient(token string) *Client {
	return &Client{
		http:    &http.Client{Timeout: 30 * time.Second},
		token:   token,
		baseURL: defaultBaseURL,
	}
}

// Fetch 拉取目标内容的全部数据（正文 + 评论）。
func (c *Client) Fetch(target *urlparse.Target) (*Content, error) {
	switch target.Kind {
	case urlparse.KindIssue, urlparse.KindPull:
		return c.fetchIssueOrPull(target)
	case urlparse.KindDiscussion:
		return c.fetchDiscussion(target)
	default:
		return nil, fmt.Errorf("unsupported kind: %s", target.Kind)
	}
}

// ---- REST (Issue / PR) ----

type restUser struct {
	Login string `json:"login"`
}

type restLabel struct {
	Name string `json:"name"`
}

type restReactions struct {
	TotalCount int `json:"total_count"`
}

// restItem 同时覆盖 issues 与 pulls 端点的响应字段（PR 特有字段为零值）。
type restItem struct {
	Title        string      `json:"title"`
	Body         string      `json:"body"`
	State        string      `json:"state"`
	CreatedAt    string      `json:"created_at"`
	User         restUser    `json:"user"`
	Labels       []restLabel `json:"labels"`
	Merged       bool        `json:"merged"`
	Additions    int         `json:"additions"`
	Deletions    int         `json:"deletions"`
	ChangedFiles int         `json:"changed_files"`
}

type restComment struct {
	User      restUser      `json:"user"`
	CreatedAt string        `json:"created_at"`
	Body      string        `json:"body"`
	Reactions restReactions `json:"reactions"`
}

func (c *Client) fetchIssueOrPull(target *urlparse.Target) (*Content, error) {
	kind := "issues"
	if target.Kind == urlparse.KindPull {
		kind = "pulls"
	}
	itemPath := fmt.Sprintf("/repos/%s/%s/%s/%d",
		url.PathEscape(target.Owner), url.PathEscape(target.Repo), kind, target.Number)

	var item restItem
	if err := c.doJSON(http.MethodGet, itemPath, nil, &item); err != nil {
		return nil, fmt.Errorf("fetch %s: %w", kind, err)
	}

	commentsPath := fmt.Sprintf("/repos/%s/%s/issues/%d/comments",
		url.PathEscape(target.Owner), url.PathEscape(target.Repo), target.Number)
	var comments []restComment
	if err := c.doJSON(http.MethodGet, commentsPath, nil, &comments); err != nil {
		return nil, fmt.Errorf("fetch comments: %w", err)
	}

	content := &Content{
		Kind:         target.Kind,
		Title:        item.Title,
		Body:         item.Body,
		Author:       item.User.Login,
		State:        item.State,
		CreatedAt:    item.CreatedAt,
		Labels:       make([]string, 0, len(item.Labels)),
		Comments:     make([]Comment, 0, len(comments)),
		Merged:       item.Merged,
		Additions:    item.Additions,
		Deletions:    item.Deletions,
		ChangedFiles: item.ChangedFiles,
	}
	for _, l := range item.Labels {
		content.Labels = append(content.Labels, l.Name)
	}
	for _, cm := range comments {
		content.Comments = append(content.Comments, Comment{
			Author:    cm.User.Login,
			CreatedAt: cm.CreatedAt,
			Body:      cm.Body,
			Reactions: cm.Reactions.TotalCount,
		})
	}
	return content, nil
}

// ---- GraphQL (Discussion) ----

// graphQLRequest 用于测试文件中的解码，字段保持公开。
type graphQLRequest struct {
	Query     string           `json:"query"`
	Variables graphQLVariables `json:"variables"`
}

type graphQLVariables struct {
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Number int    `json:"number"`
}

type graphQLResponse struct {
	Data struct {
		Repository struct {
			Discussion *graphQLDiscussion `json:"discussion"`
		} `json:"repository"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

type graphQLDiscussion struct {
	Title     string `json:"title"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
	Closed    bool   `json:"closed"`
	Author    struct {
		Login string `json:"login"`
	} `json:"author"`
	Labels struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
	Comments struct {
		TotalCount int `json:"totalCount"`
		Nodes      []struct {
			Author struct {
				Login string `json:"login"`
			} `json:"author"`
			CreatedAt string `json:"createdAt"`
			Body      string `json:"body"`
			Reactions struct {
				TotalCount int `json:"totalCount"`
			} `json:"reactions"`
		} `json:"nodes"`
	} `json:"comments"`
}

const discussionQuery = `
query($owner: String!, $repo: String!, $number: Int!) {
  repository(owner: $owner, name: $repo) {
    discussion(number: $number) {
      title
      body
      createdAt
      closed
      author { login }
      labels(first: 20) { nodes { name } }
      comments(first: 100) {
        totalCount
        nodes {
          author { login }
          createdAt
          body
          reactions(first: 20) { totalCount }
        }
      }
    }
  }
}`

func (c *Client) fetchDiscussion(target *urlparse.Target) (*Content, error) {
	if c.token == "" {
		return nil, errors.New("fetching discussions requires authentication: set GITHUB_TOKEN and retry")
	}

	reqBody := graphQLRequest{
		Query: discussionQuery,
		Variables: graphQLVariables{
			Owner:  target.Owner,
			Repo:   target.Repo,
			Number: target.Number,
		},
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal graphql request: %w", err)
	}

	var resp graphQLResponse
	if err := c.doJSON(http.MethodPost, "/graphql", payload, &resp); err != nil {
		return nil, fmt.Errorf("fetch discussion: %w", err)
	}
	if len(resp.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s", resp.Errors[0].Message)
	}
	if resp.Data.Repository.Discussion == nil {
		return nil, errors.New("graphql: discussion not found")
	}

	disc := resp.Data.Repository.Discussion
	state := "open"
	if disc.Closed {
		state = "closed"
	}

	content := &Content{
		Kind:      urlparse.KindDiscussion,
		Title:     disc.Title,
		Body:      disc.Body,
		Author:    disc.Author.Login,
		State:     state,
		CreatedAt: disc.CreatedAt,
		Labels:    make([]string, 0, len(disc.Labels.Nodes)),
		Comments:  make([]Comment, 0, len(disc.Comments.Nodes)),
	}
	for _, l := range disc.Labels.Nodes {
		content.Labels = append(content.Labels, l.Name)
	}
	for _, cm := range disc.Comments.Nodes {
		content.Comments = append(content.Comments, Comment{
			Author:    cm.Author.Login,
			CreatedAt: cm.CreatedAt,
			Body:      cm.Body,
			Reactions: cm.Reactions.TotalCount,
		})
	}
	return content, nil
}

// ---- 通用请求 ----

// doJSON 发送请求并将 JSON 响应解码到 out。body 为 nil 时发送空请求体。
func (c *Client) doJSON(method, path string, body []byte, out any) error {
	var reqBody *bytes.Reader
	if body == nil {
		reqBody = bytes.NewReader(nil)
	} else {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("create request %s %s: %w", method, path, err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var apiErr struct {
			Message string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		if resp.StatusCode == http.StatusForbidden {
			return fmt.Errorf("github api %s %s: %d %s (rate limit exceeded? try setting GITHUB_TOKEN)", method, path, resp.StatusCode, apiErr.Message)
		}
		return fmt.Errorf("github api %s %s: %d %s", method, path, resp.StatusCode, apiErr.Message)
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response from %s: %w", path, err)
		}
	}
	return nil
}
