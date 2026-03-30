package gitutil

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var cache sync.Map

// DisplayName returns a display-friendly name for a directory.
// For git worktrees, it returns "reponame(worktreename)".
// For normal directories, it returns filepath.Base(dir).
func DisplayName(dir string) string {
	if v, ok := cache.Load(dir); ok {
		return v.(string)
	}

	name := displayName(dir)
	cache.Store(dir, name)
	return name
}

func displayName(dir string) string {
	gitPath, isFile, err := findGitEntry(dir)
	if err != nil {
		return filepath.Base(dir)
	}

	if !isFile {
		return filepath.Base(dir)
	}

	repoName, worktreeName, err := parseWorktreeInfo(gitPath)
	if err != nil {
		return filepath.Base(dir)
	}

	return repoName + "(" + worktreeName + ")"
}

// findGitEntry walks up from dir looking for a .git entry.
// Returns the path to the .git entry and whether it is a file (not a directory).
func findGitEntry(dir string) (string, bool, error) {
	for {
		p := filepath.Join(dir, ".git")
		fi, err := os.Lstat(p)
		if err == nil {
			return p, !fi.IsDir(), nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false, os.ErrNotExist
		}
		dir = parent
	}
}

// parseWorktreeInfo reads a .git file and extracts the repo name and worktree name.
// The .git file contains a line like: gitdir: /path/to/repo/.git/worktrees/<name>
func parseWorktreeInfo(gitFilePath string) (repoName, worktreeName string, err error) {
	data, err := os.ReadFile(gitFilePath)
	if err != nil {
		return "", "", err
	}

	line := strings.TrimSpace(string(data))
	if !strings.HasPrefix(line, "gitdir: ") {
		return "", "", os.ErrInvalid
	}

	gitdir := strings.TrimPrefix(line, "gitdir: ")

	// Expected pattern: .../<repodir>/.git/worktrees/<worktreename>
	idx := strings.Index(gitdir, "/.git/worktrees/")
	if idx < 0 {
		return "", "", os.ErrInvalid
	}

	repoPath := gitdir[:idx]
	worktreeName = gitdir[idx+len("/.git/worktrees/"):]
	// Strip any trailing slash or path segments
	if slashIdx := strings.Index(worktreeName, "/"); slashIdx >= 0 {
		worktreeName = worktreeName[:slashIdx]
	}

	repoName = filepath.Base(repoPath)
	if repoName == "" || worktreeName == "" {
		return "", "", os.ErrInvalid
	}

	return repoName, worktreeName, nil
}

// ResetCache clears the display name cache (for testing).
func ResetCache() {
	cache = sync.Map{}
}
