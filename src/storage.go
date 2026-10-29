package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func dataFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return dataFileAt(home, filepath.Join(configDir, "pui"))
}

func dataFileAt(home, legacyDir string) (string, error) {
	root := filepath.Join(home, ".pui")
	dataDir := filepath.Join(root, "data")
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return "", err
	}
	marker := filepath.Join(dataDir, ".legacy-migrated")
	if _, err := os.Stat(marker); errors.Is(err, os.ErrNotExist) {
		for _, name := range []string{"notes.json", "folders.json", "folders"} {
			if err := copyLegacyItem(filepath.Join(legacyDir, name), filepath.Join(dataDir, name)); err != nil {
				return "", fmt.Errorf("migrate old Pui data: %w", err)
			}
		}
		if err := os.WriteFile(marker, []byte("migrated\n"), 0600); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "notes.json"), nil
}

// copyLegacyItem copies only regular files and directories, leaving the old data intact.
func copyLegacyItem(source, destination string) error {
	info, err := os.Stat(source)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err := os.Lstat(destination); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if info.IsDir() {
		if err := os.MkdirAll(destination, 0700); err != nil {
			return err
		}
		entries, err := os.ReadDir(source)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := copyLegacyItem(filepath.Join(source, entry.Name()), filepath.Join(destination, entry.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	if !info.Mode().IsRegular() {
		return nil
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil
		}
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		_ = os.Remove(destination)
		return copyErr
	}
	return closeErr
}

func loadNotes(path string) ([]Note, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return []Note{}, nil
	}
	if err != nil {
		return nil, err
	}
	var notes []Note
	if err = json.Unmarshal(b, &notes); err != nil {
		return nil, fmt.Errorf("read notes: %w", err)
	}
	if notes == nil {
		notes = []Note{}
	}
	for i := range notes {
		if notes[i].Created.IsZero() {
			notes[i].Created = notes[i].Updated
		}
		if notes[i].Updated.IsZero() {
			notes[i].Updated = notes[i].Created
		}
	}
	return notes, nil
}

func atomicJSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err = os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	if err = os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func persist(path string, notes []Note) error { return atomicJSON(path, notes) }

func validFolder(s string) bool {
	return s != "" && s != "." && s != ".." && !strings.ContainsAny(s, "/\\") && filepath.Base(s) == s && strings.TrimSpace(s) == s
}

func folderPath(root, folder string) string {
	clean := filepath.Clean(filepath.FromSlash(folder))
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return filepath.Join(root, "folders")
	}
	return filepath.Join(root, "folders", clean)
}

func newModel() (model, error) {
	p, err := dataFile()
	if err != nil {
		return model{}, err
	}
	notes, err := loadNotes(p)
	if err != nil {
		return model{}, err
	}
	root := filepath.Dir(p)
	var folders []string
	b, err := os.ReadFile(filepath.Join(root, "folders.json"))
	if err == nil {
		if err = json.Unmarshal(b, &folders); err != nil {
			return model{}, fmt.Errorf("read folders: %w", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return model{}, err
	}
	seen := map[string]bool{}
	for _, f := range folders {
		seen[f] = true
	}
	for _, n := range notes {
		f := n.Folder
		for f != "" && f != "." {
			if !seen[f] {
				folders = append(folders, f)
				seen[f] = true
			}
			f = filepath.ToSlash(filepath.Dir(filepath.FromSlash(f)))
			if f == "." {
				f = ""
			}
		}
	}
	sort.Strings(folders)
	for _, f := range folders {
		if err = os.MkdirAll(folderPath(root, f), 0700); err != nil {
			return model{}, err
		}
	}
	if err = atomicJSON(filepath.Join(root, "folders.json"), folders); err != nil {
		return model{}, err
	}
	q := textinput.New()
	q.Placeholder = "Search notes and folders…"
	q.Prompt = "⌕  "
	q.CharLimit = 120
	fi := textinput.New()
	fi.Placeholder = "Folder name"
	fi.CharLimit = 60
	return model{notes: notes, folders: folders, collapsed: map[string]bool{}, query: q, folderInput: fi, dataPath: p}, nil
}
