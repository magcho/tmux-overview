package gitutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDisplayName_NormalRepo(t *testing.T) {
	ResetCache()
	dir := t.TempDir()
	repo := filepath.Join(dir, "my-repo")
	os.MkdirAll(filepath.Join(repo, ".git"), 0o755)

	got := DisplayName(repo)
	if got != "my-repo" {
		t.Errorf("got %q, want %q", got, "my-repo")
	}
}

func TestDisplayName_Worktree(t *testing.T) {
	ResetCache()
	dir := t.TempDir()

	// Create main repo .git/worktrees/<name> structure
	mainRepo := filepath.Join(dir, "my-repo")
	worktreeGitDir := filepath.Join(mainRepo, ".git", "worktrees", "feature-branch")
	os.MkdirAll(worktreeGitDir, 0o755)

	// Create worktree directory with .git file
	worktreeDir := filepath.Join(dir, "worktrees", "feature-branch")
	os.MkdirAll(worktreeDir, 0o755)
	gitFileContent := "gitdir: " + worktreeGitDir + "\n"
	os.WriteFile(filepath.Join(worktreeDir, ".git"), []byte(gitFileContent), 0o644)

	got := DisplayName(worktreeDir)
	want := "my-repo(feature-branch)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDisplayName_NoGit(t *testing.T) {
	ResetCache()
	dir := t.TempDir()
	sub := filepath.Join(dir, "plain-dir")
	os.MkdirAll(sub, 0o755)

	got := DisplayName(sub)
	if got != "plain-dir" {
		t.Errorf("got %q, want %q", got, "plain-dir")
	}
}

func TestDisplayName_MalformedGitFile(t *testing.T) {
	ResetCache()
	dir := t.TempDir()
	repo := filepath.Join(dir, "broken")
	os.MkdirAll(repo, 0o755)
	os.WriteFile(filepath.Join(repo, ".git"), []byte("not a valid gitdir line"), 0o644)

	got := DisplayName(repo)
	if got != "broken" {
		t.Errorf("got %q, want %q", got, "broken")
	}
}

func TestDisplayName_SubdirInWorktree(t *testing.T) {
	ResetCache()
	dir := t.TempDir()

	// Create main repo structure
	mainRepo := filepath.Join(dir, "my-repo")
	worktreeGitDir := filepath.Join(mainRepo, ".git", "worktrees", "wt1")
	os.MkdirAll(worktreeGitDir, 0o755)

	// Create worktree with subdirectory
	worktreeDir := filepath.Join(dir, "wt1")
	os.MkdirAll(filepath.Join(worktreeDir, "src", "pkg"), 0o755)
	gitFileContent := "gitdir: " + worktreeGitDir + "\n"
	os.WriteFile(filepath.Join(worktreeDir, ".git"), []byte(gitFileContent), 0o644)

	// CWD is a subdirectory of the worktree
	got := DisplayName(filepath.Join(worktreeDir, "src", "pkg"))
	want := "my-repo(wt1)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDisplayName_Cache(t *testing.T) {
	ResetCache()
	dir := t.TempDir()
	repo := filepath.Join(dir, "cached-repo")
	os.MkdirAll(filepath.Join(repo, ".git"), 0o755)

	// First call
	got1 := DisplayName(repo)
	// Second call (from cache)
	got2 := DisplayName(repo)

	if got1 != got2 || got1 != "cached-repo" {
		t.Errorf("cache mismatch: got1=%q, got2=%q", got1, got2)
	}
}
