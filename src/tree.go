package main

import (
	"path/filepath"
	"sort"
	"strings"
)

func (m model) treeRows() []treeEntry {
	q := strings.ToLower(strings.TrimSpace(m.query.Value()))
	rows := []treeEntry{}
	var hasMatch func(string) bool
	hasMatch = func(folder string) bool {
		if q == "" {
			return true
		}
		if strings.Contains(strings.ToLower(folder), q) {
			return true
		}
		for _, n := range m.notes {
			if (n.Folder == folder || strings.HasPrefix(n.Folder, folder+"/")) && strings.Contains(strings.ToLower(n.Title+" "+n.Body+" "+n.Folder), q) {
				return true
			}
		}
		return false
	}
	var walk func(string, int)
	walk = func(parent string, depth int) {
		children := []string{}
		for _, f := range m.folders {
			p := filepath.ToSlash(filepath.Dir(filepath.FromSlash(f)))
			if p == "." {
				p = ""
			}
			if p == parent && hasMatch(f) {
				children = append(children, f)
			}
		}
		sort.Strings(children)
		for _, folder := range children {
			rows = append(rows, treeEntry{folder: true, path: folder, depth: depth, expanded: !m.collapsed[folder]})
			if !m.collapsed[folder] {
				walk(folder, depth+1)
			}
		}
		notes := []Note{}
		for _, n := range m.notes {
			if n.Folder == parent && (q == "" || strings.Contains(strings.ToLower(n.Title+" "+n.Body+" "+n.Folder), q)) {
				notes = append(notes, n)
			}
		}
		sort.Slice(notes, func(i, j int) bool { return strings.ToLower(notes[i].Title) < strings.ToLower(notes[j].Title) })
		for _, n := range notes {
			rows = append(rows, treeEntry{path: parent, depth: depth, note: n})
		}
	}
	if !m.collapsed[""] {
		walk("", 1)
	}
	if len(rows) == 0 && q != "" {
		return rows
	}
	root := treeEntry{folder: true, path: "", depth: 0, expanded: !m.collapsed[""]}
	return append([]treeEntry{root}, rows...)
}

func (m *model) clampCursor() {
	rows := m.treeRows()
	if m.cursor >= len(rows) {
		m.cursor = len(rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m model) selectedEntry() (treeEntry, bool) {
	rows := m.treeRows()
	if m.cursor < 0 || m.cursor >= len(rows) {
		return treeEntry{}, false
	}
	return rows[m.cursor], true
}

func (m model) folderForCurrent() string {
	e, ok := m.selectedEntry()
	if !ok {
		return ""
	}
	if e.folder {
		return e.path
	}
	return e.note.Folder
}

func (m *model) collapseOrParent() {
	e, ok := m.selectedEntry()
	if !ok {
		return
	}
	if e.folder && e.expanded {
		m.collapsed[e.path] = true
		return
	}
	parent := e.path
	if e.folder {
		parent = filepath.ToSlash(filepath.Dir(filepath.FromSlash(e.path)))
		if parent == "." {
			parent = ""
		}
	}
	rows := m.treeRows()
	for i, row := range rows {
		if row.folder && row.path == parent {
			m.cursor = i
			return
		}
	}
}
