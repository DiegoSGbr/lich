package project

import (
	"slices"
	"testing"
)

func TestParsePRDetail(t *testing.T) {
	t.Run("open PR with mixed checks", func(t *testing.T) {
		out := []byte(`{
			"number": 88,
			"url": "https://github.com/omartelo/lich/pull/88",
			"state": "OPEN",
			"title": "feat: view and merge pull requests",
			"body": "makes the badge actionable",
			"isDraft": false,
			"mergeable": "MERGEABLE",
			"baseRefName": "main",
			"headRefName": "quiet-willow",
			"statusCheckRollup": [
				{"status": "COMPLETED", "conclusion": "SUCCESS"},
				{"state": "SUCCESS"}
			]
		}`)
		pr, err := parsePRDetail(out)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pr == nil {
			t.Fatal("expected a detail, got nil")
		}
		if pr.Number != 88 || pr.Title != "feat: view and merge pull requests" {
			t.Errorf("wrong header: %+v", pr)
		}
		if pr.Mergeable != "MERGEABLE" || pr.BaseRefName != "main" || pr.HeadRefName != "quiet-willow" {
			t.Errorf("wrong refs/mergeable: %+v", pr)
		}
		if pr.Checks.Passed != 2 || pr.Checks.Total != 2 || pr.Checks.Failed != 0 {
			t.Errorf("wrong checks: %+v", pr.Checks)
		}
	})

	t.Run("draft flag survives", func(t *testing.T) {
		pr, err := parsePRDetail([]byte(`{"number":9,"state":"OPEN","isDraft":true}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pr == nil || !pr.IsDraft {
			t.Errorf("expected draft detail, got %+v", pr)
		}
	})

	t.Run("non-open PR is hidden", func(t *testing.T) {
		pr, err := parsePRDetail([]byte(`{"number":88,"state":"MERGED","url":"x"}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pr != nil {
			t.Errorf("merged PR should yield nil, got %+v", pr)
		}
	})

	t.Run("no PR object is hidden", func(t *testing.T) {
		pr, err := parsePRDetail([]byte(`{}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if pr != nil {
			t.Errorf("empty payload should yield nil, got %+v", pr)
		}
	})

	t.Run("malformed JSON errors", func(t *testing.T) {
		if _, err := parsePRDetail([]byte(`{not json`)); err == nil {
			t.Error("expected a decode error")
		}
	})
}

func TestReduceChecks(t *testing.T) {
	tests := []struct {
		name  string
		items []checkItem
		want  ChecksRollup
	}{
		{"none", nil, ChecksRollup{}},
		{
			"completed check runs pass and fail",
			[]checkItem{
				{Status: "COMPLETED", Conclusion: "SUCCESS"},
				{Status: "COMPLETED", Conclusion: "FAILURE"},
			},
			ChecksRollup{Passed: 1, Failed: 1, Total: 2},
		},
		{
			"neutral and skipped count as pass",
			[]checkItem{
				{Status: "COMPLETED", Conclusion: "NEUTRAL"},
				{Status: "COMPLETED", Conclusion: "SKIPPED"},
			},
			ChecksRollup{Passed: 2, Total: 2},
		},
		{
			"in-flight check run is pending",
			[]checkItem{{Status: "IN_PROGRESS"}, {Status: "QUEUED"}},
			ChecksRollup{Pending: 2, Total: 2},
		},
		{
			"legacy status contexts map by state",
			[]checkItem{
				{State: "SUCCESS"},
				{State: "FAILURE"},
				{State: "ERROR"},
				{State: "PENDING"},
			},
			ChecksRollup{Passed: 1, Failed: 2, Pending: 1, Total: 4},
		},
		{
			"mixed shapes sum together",
			[]checkItem{
				{Status: "COMPLETED", Conclusion: "SUCCESS"},
				{State: "SUCCESS"},
				{Status: "IN_PROGRESS"},
			},
			ChecksRollup{Passed: 2, Pending: 1, Total: 3},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reduceChecks(tt.items); got != tt.want {
				t.Errorf("reduceChecks() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestMergeArgs(t *testing.T) {
	// An unrecognised method must be refused before any gh shell-out.
	if _, err := mergeArgs("force", "", ""); err == nil {
		t.Error("expected an error for an unknown merge method")
	}

	tests := []struct {
		name    string
		method  string
		subject string
		body    string
		want    []string
	}{
		{"quick squash", "squash", "", "", []string{"pr", "merge", "--squash"}},
		{"quick merge", "merge", "", "", []string{"pr", "merge", "--merge"}},
		{"quick rebase", "rebase", "", "", []string{"pr", "merge", "--rebase"}},
		{
			"squash with edited message",
			"squash", "title (#1)", "details",
			[]string{"pr", "merge", "--squash", "--subject", "title (#1)", "--body", "details"},
		},
		{
			"empty body still passes both flags",
			"merge", "subject only", "",
			[]string{"pr", "merge", "--merge", "--subject", "subject only", "--body", ""},
		},
		{
			"rebase ignores an edited message",
			"rebase", "unused", "unused",
			[]string{"pr", "merge", "--rebase"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mergeArgs(tt.method, tt.subject, tt.body)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("mergeArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNoPullRequest(t *testing.T) {
	tests := []struct {
		stderr string
		want   bool
	}{
		{`no pull requests found for branch "quiet-willow"`, true},
		{"no pull requests found for current branch", true},
		{"", false},
		{"could not find any commits between main and X", false},
		{"gh: command not found", false},
	}
	for _, tt := range tests {
		if got := isNoPullRequest(tt.stderr); got != tt.want {
			t.Errorf("isNoPullRequest(%q) = %v, want %v", tt.stderr, got, tt.want)
		}
	}
}

func TestGhError(t *testing.T) {
	if got := ghError("  boom  ", errTest); got != "boom" {
		t.Errorf("stderr should win and be trimmed, got %q", got)
	}
	if got := ghError("   ", errTest); got != "sentinel" {
		t.Errorf("empty stderr should fall back to the error, got %q", got)
	}
}

// errTest is a fixed error for ghError's fallback branch.
var errTest = testError("sentinel")

type testError string

func (e testError) Error() string { return string(e) }
