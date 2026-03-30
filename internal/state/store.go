package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Store manages per-pane state files in a directory.
type Store struct {
	dir string
}

// DefaultStateDir returns the default state directory path.
func DefaultStateDir() string {
	if tmp := os.Getenv("TMPDIR"); tmp != "" {
		return filepath.Join(tmp, "tov")
	}
	return filepath.Join(os.TempDir(), "tov")
}

// NewStore creates a Store with the given directory.
// If dir is empty, DefaultStateDir() is used.
func NewStore(dir string) *Store {
	if dir == "" {
		dir = DefaultStateDir()
	}
	return &Store{dir: dir}
}

// Dir returns the state directory path.
func (s *Store) Dir() string {
	return s.dir
}

// EnsureDir creates the state directory if it does not exist.
func (s *Store) EnsureDir() error {
	return os.MkdirAll(s.dir, 0700)
}

// paneFilename converts a pane ID like "%23" to a safe filename "_23.json".
func paneFilename(paneID string) string {
	safe := strings.ReplaceAll(paneID, "%", "_")
	return safe + ".json"
}

// paneIDFromFilename converts a filename like "_23.json" back to a pane ID "%23".
func paneIDFromFilename(name string) string {
	name = strings.TrimSuffix(name, ".json")
	return strings.ReplaceAll(name, "_", "%")
}

// Write atomically writes a PaneState to its state file.
func (s *Store) Write(ps PaneState) error {
	if err := s.EnsureDir(); err != nil {
		return fmt.Errorf("ensuring state dir: %w", err)
	}

	data, err := json.Marshal(ps)
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	target := filepath.Join(s.dir, paneFilename(ps.PaneID))
	tmp := target + ".tmp"

	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := os.Rename(tmp, target); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("renaming state file: %w", err)
	}

	return nil
}

// Read reads a single pane's state. Returns the state, whether the file exists, and any error.
func (s *Store) Read(paneID string) (PaneState, bool, error) {
	path := filepath.Join(s.dir, paneFilename(paneID))
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return PaneState{}, false, nil
		}
		return PaneState{}, false, fmt.Errorf("reading state file: %w", err)
	}

	var ps PaneState
	if err := json.Unmarshal(data, &ps); err != nil {
		return PaneState{}, false, fmt.Errorf("parsing state file: %w", err)
	}
	return ps, true, nil
}

// ListAll reads all state files in the directory.
func (s *Store) ListAll() ([]PaneState, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading state dir: %w", err)
	}

	var states []PaneState
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") || strings.HasSuffix(e.Name(), ".tmp") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			continue
		}
		var ps PaneState
		if err := json.Unmarshal(data, &ps); err != nil {
			continue
		}
		states = append(states, ps)
	}
	return states, nil
}

// Remove deletes a pane's state file.
func (s *Store) Remove(paneID string) error {
	path := filepath.Join(s.dir, paneFilename(paneID))
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing state file: %w", err)
	}
	return nil
}

// RemoveStale removes state files for panes not in the live set.
// Returns the number of removed files.
func (s *Store) RemoveStale(livePaneIDs map[string]bool) int {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return 0
	}

	removed := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") || strings.HasSuffix(e.Name(), ".tmp") {
			continue
		}
		paneID := paneIDFromFilename(e.Name())
		if !livePaneIDs[paneID] {
			os.Remove(filepath.Join(s.dir, e.Name()))
			removed++
		}
	}
	return removed
}
