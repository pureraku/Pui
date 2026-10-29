package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func TestEnterTogglesRootAndFolder(t *testing.T) {
	m := model{
		folders:     []string{"Work"},
		collapsed:   map[string]bool{},
		query:       textinput.New(),
		folderInput: textinput.New(),
	}
	pressEnter := func() {
		t.Helper()
		next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m = next.(model)
	}

	pressEnter()
	if !m.collapsed[""] || len(m.treeRows()) != 1 {
		t.Fatal("Enter should collapse Root and hide its child")
	}
	pressEnter()
	if m.collapsed[""] || len(m.treeRows()) != 2 {
		t.Fatal("Enter should expand Root and show its child")
	}
	m.cursor = 1
	pressEnter()
	if !m.collapsed["Work"] {
		t.Fatal("Enter should collapse the selected child folder")
	}
	pressEnter()
	if m.collapsed["Work"] {
		t.Fatal("Enter should expand the selected child folder")
	}
}

func TestDeleteConfirmationIsVisible(t *testing.T) {
	m := model{
		screen:      deleteScreen,
		cursor:      1,
		notes:       []Note{{ID: "n1", Title: "Example"}},
		collapsed:   map[string]bool{},
		query:       textinput.New(),
		folderInput: textinput.New(),
		width:       80,
		height:      24,
	}
	view := m.View()
	for _, want := range []string{"Delete note", "Enter/y confirm", "n/Esc cancel"} {
		if !strings.Contains(view, want) {
			t.Errorf("delete confirmation view missing %q", want)
		}
	}
}

func TestResponsiveViewKeepsNavigationHints(t *testing.T) {
	for _, size := range [][2]int{{38, 12}, {80, 24}, {140, 40}} {
		m := model{
			notes:       []Note{},
			collapsed:   map[string]bool{},
			query:       textinput.New(),
			folderInput: textinput.New(),
			width:       size[0],
			height:      size[1],
		}
		view := m.View()
		if !strings.Contains(view, "enter") || !strings.Contains(view, "j/k") {
			t.Errorf("view at %dx%d lost navigation hints", size[0], size[1])
		}
	}
}

func TestLegacyDataMigrationCopiesOnce(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	legacy := filepath.Join(base, "Library", "Application Support", "pui")
	if err := os.MkdirAll(filepath.Join(legacy, "folders", "Work"), 0700); err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{
		filepath.Join(legacy, "notes.json"):                  "[]",
		filepath.Join(legacy, "folders.json"):                `["Work"]`,
		filepath.Join(legacy, "folders", "Work", "keep.txt"): "keep",
	} {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}

	dataPath, err := dataFileAt(home, legacy)
	if err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"notes.json", "folders.json", filepath.Join("folders", "Work", "keep.txt")} {
		if _, err := os.Stat(filepath.Join(filepath.Dir(dataPath), rel)); err != nil {
			t.Errorf("expected migrated %s: %v", rel, err)
		}
	}
	if err := os.Remove(dataPath); err != nil {
		t.Fatal(err)
	}
	if _, err := dataFileAt(home, legacy); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dataPath); !os.IsNotExist(err) {
		t.Fatalf("migration should not restore a file deleted after migration; stat error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(legacy, "notes.json")); err != nil {
		t.Fatalf("migration should leave old data in place: %v", err)
	}
}
