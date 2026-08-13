// Package commentfilter 从评论中筛选高价值评论（it works 双确认 + reactions 排序）。
package commentfilter

import (
	"regexp"
	"sort"
	"strings"

	"github.com/bigwhite/issue2md/internal/github"
)

// maxComments 是精选评论的上限。
const maxComments = 3

// englishPattern 匹配英文关键词（词边界，不区分大小写）。
// 注意 alternation 顺序：较长的短语放在前面，避免被短词前缀吞掉。
var englishPattern = regexp.MustCompile(`(?i)\b(?:it works|worked for me|worked|solved|fixed)\b`)

// chineseKeywords 是中文关键词（substring 匹配）。
var chineseKeywords = []string{"解决了", "有效", "可以了", "成功了", "搞定了"}

// Select 从评论中选出最多 3 条高价值评论：
//  1. it works 双确认：正文含关键词 且 reactions 总数 > 0，按 reactions 降序取前 3；
//  2. 兜底：无 it works 评论时，取有 reactions 的评论中 reactions 最高的前 3 条；
//  3. 边界：不足 3 条有多少输出多少，一条都没有返回 nil。
func Select(comments []github.Comment) []github.Comment {
	if len(comments) == 0 {
		return nil
	}

	var works []github.Comment      // it works 评论（双确认）
	var candidates []github.Comment // 兜底候选（有 reactions 的评论）
	for _, cm := range comments {
		if cm.Reactions > 0 {
			candidates = append(candidates, cm)
		}
		if isItWorks(cm) {
			works = append(works, cm)
		}
	}

	pool := works
	if len(pool) == 0 {
		pool = candidates
	}
	if len(pool) == 0 {
		return nil
	}

	// 稳定排序：reactions 相同时保持原始顺序。
	sort.SliceStable(pool, func(i, j int) bool {
		return pool[i].Reactions > pool[j].Reactions
	})
	if len(pool) > maxComments {
		pool = pool[:maxComments]
	}
	return pool
}

// isItWorks 判断评论是否为 it works 类型（关键词 + reactions 双确认）。
func isItWorks(cm github.Comment) bool {
	if cm.Reactions <= 0 {
		return false
	}
	if englishPattern.MatchString(cm.Body) {
		return true
	}
	body := strings.ToLower(cm.Body)
	for _, kw := range chineseKeywords {
		if strings.Contains(body, kw) {
			return true
		}
	}
	return false
}
