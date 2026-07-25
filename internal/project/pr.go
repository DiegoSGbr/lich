package project

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/omartelo/lich/internal/winexec"
)

// gh network calls are capped so a slow forge or a hung auth prompt never
// wedges the dock panel; merge and create get more room because they push and
// mutate on the remote.
const (
	prDetailTimeout = 8 * time.Second
	prMergeTimeout  = 30 * time.Second
	prCreateTimeout = 20 * time.Second
)

// prViewFields is the gh `pr view --json` selection backing the Pulls panel.
const prViewFields = "number,url,state,title,body,isDraft,mergeable,baseRefName,headRefName,statusCheckRollup,changedFiles"

// PRDetail is the full view of a branch's open pull request — richer than the
// footer badge's PullRequest — driving the dock's Pulls panel: the title, body,
// CI rollup and mergeability gate the merge affordance.
type PRDetail struct {
	Number       int          `json:"number"`
	URL          string       `json:"url"`
	State        string       `json:"state"`
	Title        string       `json:"title"`
	Body         string       `json:"body"`
	IsDraft      bool         `json:"isDraft"`
	Mergeable    string       `json:"mergeable"` // gh: MERGEABLE | CONFLICTING | UNKNOWN
	BaseRefName  string       `json:"baseRefName"`
	HeadRefName  string       `json:"headRefName"`
	ChangedFiles int          `json:"changedFiles"`
	Checks       ChecksRollup `json:"checks"`
}

// ChecksRollup collapses gh's statusCheckRollup array into the counts the panel
// shows. Total is every check, so "all green" is Passed == Total && Total > 0.
type ChecksRollup struct {
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Pending int `json:"pending"`
	Total   int `json:"total"`
}

// ghPRView mirrors the requested gh payload; statusCheckRollup is reduced to
// ChecksRollup inside parsePRDetail and never leaves this package raw.
type ghPRView struct {
	Number            int         `json:"number"`
	URL               string      `json:"url"`
	State             string      `json:"state"`
	Title             string      `json:"title"`
	Body              string      `json:"body"`
	IsDraft           bool        `json:"isDraft"`
	Mergeable         string      `json:"mergeable"`
	BaseRefName       string      `json:"baseRefName"`
	HeadRefName       string      `json:"headRefName"`
	ChangedFiles      int         `json:"changedFiles"`
	StatusCheckRollup []checkItem `json:"statusCheckRollup"`
}

// checkItem is one statusCheckRollup entry. gh emits two shapes: a CheckRun
// (Actions/apps) carries status+conclusion; a StatusContext (legacy commit
// statuses) carries state. reduceChecks reads whichever is populated.
type checkItem struct {
	Status     string `json:"status"`     // CheckRun: QUEUED|IN_PROGRESS|COMPLETED|…
	Conclusion string `json:"conclusion"` // CheckRun: SUCCESS|FAILURE|NEUTRAL|SKIPPED|…
	State      string `json:"state"`      // StatusContext: SUCCESS|FAILURE|PENDING|ERROR
}

// PullRequestDetail returns the open pull request for the path's current branch
// in full, or nil when the branch has no open PR — gh reports none, or the PR is
// merged/closed, which the OPEN-only gate treats the same as the badge does. A
// real failure (gh missing, not a GitHub repo) yields an error so the panel can
// tell "no PR" apart from "could not look up".
func (s *Service) PullRequestDetail(path string) (*PRDetail, error) {
	ctx, cancel := context.WithTimeout(context.Background(), prDetailTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", "pr", "view", "--json", prViewFields)
	cmd.Dir = path
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	winexec.Hide(cmd)
	out, err := cmd.Output()
	if err != nil {
		if isNoPullRequest(stderr.String()) {
			return nil, nil
		}
		return nil, fmt.Errorf("gh pr view: %s", ghError(stderr.String(), err))
	}
	return parsePRDetail(out)
}

// parsePRDetail decodes gh's JSON and reduces the check rollup. It returns nil
// for a non-OPEN PR — gh still reports a merged/closed branch PR, but the panel
// wants only an actionable one — matching parsePullRequest's contract.
func parsePRDetail(out []byte) (*PRDetail, error) {
	var v ghPRView
	if err := json.Unmarshal(out, &v); err != nil {
		return nil, fmt.Errorf("decode gh pr view: %w", err)
	}
	if v.Number == 0 || v.State != "OPEN" {
		return nil, nil
	}
	return &PRDetail{
		Number:       v.Number,
		URL:          v.URL,
		State:        v.State,
		Title:        v.Title,
		Body:         v.Body,
		IsDraft:      v.IsDraft,
		Mergeable:    v.Mergeable,
		BaseRefName:  v.BaseRefName,
		HeadRefName:  v.HeadRefName,
		ChangedFiles: v.ChangedFiles,
		Checks:       reduceChecks(v.StatusCheckRollup),
	}, nil
}

// reduceChecks collapses gh's mixed CheckRun/StatusContext rollup into counts. A
// CheckRun is pending until its status is COMPLETED, then passes unless the
// conclusion is a failure; a StatusContext maps its state directly. Anything
// unrecognised counts as pending, so an in-flight check never reads as passed.
func reduceChecks(items []checkItem) ChecksRollup {
	r := ChecksRollup{Total: len(items)}
	for _, it := range items {
		switch {
		case it.Status != "" && it.Status != "COMPLETED":
			r.Pending++
		case it.Conclusion != "":
			if isFailureConclusion(it.Conclusion) {
				r.Failed++
			} else {
				r.Passed++
			}
		case it.State == "SUCCESS":
			r.Passed++
		case it.State == "FAILURE" || it.State == "ERROR":
			r.Failed++
		default:
			r.Pending++
		}
	}
	return r
}

func isFailureConclusion(c string) bool {
	switch c {
	case "FAILURE", "TIMED_OUT", "CANCELLED", "ACTION_REQUIRED", "STARTUP_FAILURE", "STALE":
		return true
	}
	return false
}

// mergeMethods maps each accepted method to its gh flag, and doubles as the
// allow-list: an unknown method is rejected before any shell-out.
var mergeMethods = map[string]string{
	"squash": "--squash",
	"merge":  "--merge",
	"rebase": "--rebase",
}

// MergePullRequest merges the path's branch PR on GitHub with the given method
// (squash|merge|rebase). A non-empty subject overrides the commit message (the
// dock's "edit message" flow); an empty subject leaves gh's default, the quick
// merge. It does not pass --delete-branch: lich's worktree removal owns that
// cleanup, and deleting a checked-out worktree's branch is trouble. gh's stderr
// is surfaced on failure (not mergeable, branch protection, gh missing).
func (s *Service) MergePullRequest(path, method, subject, body string) error {
	args, err := mergeArgs(method, subject, body)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), prMergeTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", args...)
	cmd.Dir = path
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	winexec.Hide(cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh pr merge: %s", ghError(stderr.String(), err))
	}
	return nil
}

// mergeArgs builds the gh argument list for a merge, rejecting an unknown method
// before any shell-out. The subject override applies only to squash and merge
// commits — rebase replays the branch's own commits, so gh rejects a message
// there — and an empty subject means gh's default message. body rides along with
// a subject and may itself be empty.
func mergeArgs(method, subject, body string) ([]string, error) {
	flag, ok := mergeMethods[method]
	if !ok {
		return nil, fmt.Errorf("unknown merge method %q", method)
	}
	args := []string{"pr", "merge", flag}
	if subject != "" && method != "rebase" {
		args = append(args, "--subject", subject, "--body", body)
	}
	return args, nil
}

// CreatePullRequest opens GitHub's "new pull request" page in the browser for
// the path's branch (gh pushes the branch first when it has no upstream).
// Deliberately the web flow, not an in-app form — GitHub's page already owns the
// title, body, reviewers and template. Returns once the browser is launched.
func (s *Service) CreatePullRequest(path string) error {
	ctx, cancel := context.WithTimeout(context.Background(), prCreateTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", "pr", "create", "--web")
	cmd.Dir = path
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	winexec.Hide(cmd)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh pr create: %s", ghError(stderr.String(), err))
	}
	return nil
}

// PullRequestDiff returns the unified diff of the path's branch pull request —
// every change the PR would merge into its base, as GitHub computes it — for the
// Pulls screen's "Files changed" tab. --color never keeps the output plain so
// the frontend's parseDiff reads it. An empty string with no error means the
// branch has no open PR (nothing to show).
func (s *Service) PullRequestDiff(path string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), prDetailTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", "pr", "diff", "--color", "never")
	cmd.Dir = path
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	winexec.Hide(cmd)
	out, err := cmd.Output()
	if err != nil {
		if isNoPullRequest(stderr.String()) {
			return "", nil
		}
		return "", fmt.Errorf("gh pr diff: %s", ghError(stderr.String(), err))
	}
	return string(out), nil
}

// isNoPullRequest recognises gh's "no PR for this branch" message — the one
// failure that means an empty panel rather than a real error.
func isNoPullRequest(stderr string) bool {
	return strings.Contains(strings.ToLower(stderr), "no pull requests found")
}

// ghError prefers gh's own stderr (it names the cause — not mergeable, auth,
// missing repo) and falls back to the bare exec error when stderr is empty.
func ghError(stderr string, err error) string {
	if msg := strings.TrimSpace(stderr); msg != "" {
		return msg
	}
	return err.Error()
}
