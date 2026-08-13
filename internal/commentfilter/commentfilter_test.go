package commentfilter

import (
	"testing"

	"github.com/bigwhite/issue2md/internal/github"
)

func TestSelect(t *testing.T) {
	comment := func(body string, reactions int) github.Comment {
		return github.Comment{Author: "author", Body: body, Reactions: reactions}
	}

	tests := []struct {
		name     string
		comments []github.Comment
		want     []string // 期望选中的评论 body 列表
	}{
		{
			name: "it works double-confirmed comments selected in reactions order",
			comments: []github.Comment{
				comment("Try upgrading, it works for me.", 5),
				comment("This solved my problem.", 9),
				comment("No idea what is happening.", 0),
				comment("Fixed the issue by clearing cache.", 2),
			},
			want: []string{"This solved my problem.", "Try upgrading, it works for me.", "Fixed the issue by clearing cache."},
		},
		{
			name: "keyword without reactions is not selected",
			comments: []github.Comment{
				comment("it works but I have no reactions.", 0),
				comment("This solved my issue.", 1),
			},
			want: []string{"This solved my issue."},
		},
		{
			name: "reactions without keyword is not selected",
			comments: []github.Comment{
				comment("Interesting approach.", 100),
				comment("Solved it!", 3),
			},
			want: []string{"Solved it!"},
		},
		{
			name: "at most three comments selected",
			comments: []github.Comment{
				comment("Worked for me.", 1),
				comment("solved", 2),
				comment("Fixed", 3),
				comment("It works", 4),
				comment("搞定了", 5),
			},
			want: []string{"搞定了", "It works", "Fixed"},
		},
		{
			name: "fallback to top reactions when no it works comments",
			comments: []github.Comment{
				comment("First comment.", 1),
				comment("Second comment.", 10),
				comment("Third comment.", 5),
				comment("Fourth comment.", 2),
			},
			want: []string{"Second comment.", "Third comment.", "Fourth comment."},
		},
		{
			name: "fewer than three comments returns what exists",
			comments: []github.Comment{
				comment("It works.", 7),
			},
			want: []string{"It works."},
		},
		{
			name:     "empty comments returns empty",
			comments: nil,
			want:     nil,
		},
		{
			name: "case insensitive keyword match",
			comments: []github.Comment{
				comment("IT WORKS GREAT!", 3),
			},
			want: []string{"IT WORKS GREAT!"},
		},
		{
			name: "chinese keyword 有效",
			comments: []github.Comment{
				comment("这个方法有效", 4),
			},
			want: []string{"这个方法有效"},
		},
		{
			name: "fallback empty when no reactions at all",
			comments: []github.Comment{
				comment("Just a note.", 0),
				comment("Another note.", 0),
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Select(tt.comments)
			if len(got) != len(tt.want) {
				t.Fatalf("Select() returned %d comments, want %d: %+v", len(got), len(tt.want), got)
			}
			for i, want := range tt.want {
				if got[i].Body != want {
					t.Errorf("Select()[%d].Body = %q, want %q", i, got[i].Body, want)
				}
			}
		})
	}
}
