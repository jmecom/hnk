package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Repository struct {
	Path string
}

func NewRepository(path string) *Repository {
	return &Repository{Path: path}
}

func (r *Repository) execGit(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", args...)
	if r.Path != "" {
		cmd.Dir = r.Path
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, stderr.String())
	}

	return stdout.String(), nil
}

func (r *Repository) GetDiff(ctx context.Context, staged bool, paths ...string) (string, error) {
	args := []string{"diff", "--no-color", "-U3"}
	if staged {
		args = append(args, "--cached")
	}
	if len(paths) > 0 {
		args = append(args, "--")
		args = append(args, paths...)
	}
	return r.execGit(ctx, args...)
}

func (r *Repository) GetDiffAgainstRef(ctx context.Context, ref string, paths ...string) (string, error) {
	args := []string{"diff", "--no-color", "-U3", ref}
	if len(paths) > 0 {
		args = append(args, "--")
		args = append(args, paths...)
	}
	return r.execGit(ctx, args...)
}

func (r *Repository) GetDiffBetweenRefs(ctx context.Context, from, to string, paths ...string) (string, error) {
	args := []string{"diff", "--no-color", "-U3", from, to}
	if len(paths) > 0 {
		args = append(args, "--")
		args = append(args, paths...)
	}
	return r.execGit(ctx, args...)
}

func (r *Repository) GetLog(ctx context.Context, n int) (string, error) {
	return r.execGit(ctx, "log", "--oneline", fmt.Sprintf("-%d", n))
}

func (r *Repository) GetCurrentBranch(ctx context.Context) (string, error) {
	out, err := r.execGit(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func (r *Repository) GetStatus(ctx context.Context) (string, error) {
	return r.execGit(ctx, "status", "--short")
}

func (r *Repository) IsRepo() bool {
	ctx := context.Background()
	_, err := r.execGit(ctx, "rev-parse", "--git-dir")
	return err == nil
}

func (r *Repository) GetCommitDiff(ctx context.Context, commit string, paths ...string) (string, error) {
	args := []string{"show", "--no-color", "-U3", "--format=", commit}
	if len(paths) > 0 {
		args = append(args, "--")
		args = append(args, paths...)
	}
	return r.execGit(ctx, args...)
}

func (r *Repository) IsValidRef(ctx context.Context, ref string) bool {
	_, err := r.execGit(ctx, "rev-parse", "--verify", ref+"^{commit}")
	return err == nil
}
