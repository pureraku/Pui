package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func (m model) Init() tea.Cmd { return textinput.Blink }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case editResultMsg:
		if msg.err != nil {
			m.message = "Neovim edit failed: " + msg.err.Error()
			return m, nil
		}
		m.saveEditedNote(msg.title, msg.body)
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.creatingFolder || m.renamingFolder {
			switch msg.String() {
			case "esc":
				m.creatingFolder = false
				m.renamingFolder = false
				m.folderInput.Blur()
				m.folderInput.SetValue("")
				return m, nil
			case "enter":
				if m.renamingFolder {
					m.renameFolder()
				} else {
					m.createFolder()
				}
				return m, nil
			}
			var c tea.Cmd
			m.folderInput, c = m.folderInput.Update(msg)
			return m, c
		}
		if m.query.Focused() {
			if msg.String() == "esc" {
				m.query.Blur()
				return m, nil
			}
			var c tea.Cmd
			m.query, c = m.query.Update(msg)
			m.cursor = 0
			return m, c
		}
		if m.screen == deleteScreen {
			switch msg.String() {
			case "y", "enter":
				if m.pendingFolderDelete != "" {
					m.deleteFolder()
				} else {
					m.removeSelected()
				}
			case "n", "esc":
				m.pendingFolderDelete = ""
				m.screen = listScreen
			}
			return m, nil
		}
		m.clampCursor()
		entry, hasEntry := m.selectedEntry()
		switch msg.String() {
		case "q":
			return m, tea.Quit
		case "g":
			m.collapsed[""] = false
			m.cursor = 0
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.treeRows())-1 {
				m.cursor++
			}
		case "left", "h", "backspace":
			m.collapseOrParent()
		case "right", "l":
			if hasEntry && entry.folder {
				m.collapsed[entry.path] = false
			}
		case "enter":
			if hasEntry && entry.folder {
				m.collapsed[entry.path] = entry.expanded
			} else if hasEntry {
				m.startNoteEdit(entry.note)
				return m, m.launchEditor(entry.note)
			}
		case "e":
			if hasEntry && !entry.folder {
				m.startNoteEdit(entry.note)
				return m, m.launchEditor(entry.note)
			}
		case "ctrl+n", "n":
			folder := m.folderForCurrent()
			m.startNote(folder)
			return m, m.launchEditor(Note{Folder: folder})
		case "d":
			if hasEntry && (!entry.folder || entry.path != "") {
				m.screen = deleteScreen
				if entry.folder {
					m.pendingFolderDelete = entry.path
				} else {
					m.pendingFolderDelete = ""
				}
				return m, nil
			}
		case "r":
			if hasEntry && entry.folder && entry.path != "" {
				m.startRenameFolder(entry.path)
				return m, textinput.Blink
			}
		case "x":
			if hasEntry && !entry.folder {
				m.toggleChecked(entry.note.ID)
			}
		case "m":
			if hasEntry && !entry.folder {
				m.movingID = entry.note.ID
				m.message = "Move selected note: navigate to a folder, then press p to place it"
			}
		case "p":
			if m.movingID != "" {
				m.placeMovingNote(m.folderForCurrent())
			}
		case "f":
			m.startFolder(m.folderForCurrent())
			return m, textinput.Blink
		case "/":
			m.query.Focus()
			return m, textinput.Blink
		case "esc":
			m.query.SetValue("")
			m.cursor = 0
			m.movingID = ""
			m.message = ""
		}
	}
	return m, nil
}

func (m *model) startNote(folder string) { m.editingID = ""; m.editingFolder = folder; m.message = "" }

func (m *model) startNoteEdit(n Note) { m.editingID = n.ID; m.editingFolder = n.Folder; m.message = "" }

func (m *model) launchEditor(n Note) tea.Cmd {
	binary, err := exec.LookPath("nvim")
	if err != nil {
		m.message = "Neovim was not found on PATH"
		return nil
	}
	tmp, err := os.CreateTemp("", "pui-note-*.md")
	if err != nil {
		m.message = err.Error()
		return nil
	}
	content := "# " + n.Title + "\n" + n.Body
	if _, err = tmp.WriteString(content); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		m.message = err.Error()
		return nil
	}
	path := tmp.Name()
	if err = tmp.Close(); err != nil {
		os.Remove(path)
		m.message = err.Error()
		return nil
	}
	cmd := exec.Command(binary, path)
	return tea.ExecProcess(cmd, func(runErr error) tea.Msg {
		defer os.Remove(path)
		b, readErr := os.ReadFile(path)
		if runErr != nil {
			return editResultMsg{err: runErr}
		}
		if readErr != nil {
			return editResultMsg{err: readErr}
		}
		lines := strings.SplitN(string(b), "\n", 2)
		head := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[0]), "#"))
		body := ""
		if len(lines) > 1 {
			body = lines[1]
		}
		if head == "" {
			head = firstLine(body)
		}
		return editResultMsg{title: head, body: body}
	})
}

func (m *model) saveEditedNote(title, body string) {
	if title == "" {
		title = firstLine(body)
	}
	if title == "" {
		m.message = "Note left empty; nothing saved"
		return
	}
	now := time.Now()
	n := Note{ID: m.editingID, Title: title, Body: body, Folder: m.editingFolder, Created: now, Updated: now}
	if n.ID != "" {
		for i := range m.notes {
			if m.notes[i].ID == n.ID {
				n.Created = m.notes[i].Created
				n.Checked = m.notes[i].Checked
				m.notes[i] = n
				goto saved
			}
		}
	}
	n.ID = fmt.Sprintf("%d", now.UnixNano())
	m.notes = append(m.notes, n)
saved:
	sort.Slice(m.notes, func(i, j int) bool { return m.notes[i].Updated.After(m.notes[j].Updated) })
	m.message = ""
	if err := persist(m.dataPath, m.notes); err != nil {
		m.message = "Could not save note: " + err.Error()
	}
}

func (m *model) startFolder(parent string) {
	m.creatingFolder = true
	m.folderParent = parent
	m.folderInput.SetValue("")
	m.folderInput.Focus()
}

func (m *model) createFolder() {
	name := strings.TrimSpace(m.folderInput.Value())
	m.creatingFolder = false
	m.folderInput.Blur()
	m.folderInput.SetValue("")
	if !validFolder(name) {
		m.message = "Folder names cannot be empty or contain / or \\"
		return
	}
	path := name
	if m.folderParent != "" {
		path = m.folderParent + "/" + name
	}
	for _, f := range m.folders {
		if f == path {
			m.message = "That folder already exists"
			return
		}
	}
	if err := os.MkdirAll(folderPath(filepath.Dir(m.dataPath), path), 0700); err != nil {
		m.message = "Could not create folder: " + err.Error()
		return
	}
	m.folders = append(m.folders, path)
	sort.Strings(m.folders)
	if err := atomicJSON(filepath.Join(filepath.Dir(m.dataPath), "folders.json"), m.folders); err != nil {
		m.message = "Could not save folder list: " + err.Error()
		return
	}
	m.collapsed[m.folderParent] = false
	m.message = "Created folder " + path
}

func (m *model) startRenameFolder(path string) {
	m.renamingFolder = true
	m.folderTarget = path
	m.folderInput.SetValue(filepath.Base(path))
	m.folderInput.Focus()
}

func (m *model) renameFolder() {
	name := strings.TrimSpace(m.folderInput.Value())
	old := m.folderTarget
	m.renamingFolder = false
	m.folderTarget = ""
	m.folderInput.Blur()
	m.folderInput.SetValue("")
	if !validFolder(name) {
		m.message = "Folder names cannot be empty or contain / or \\"
		return
	}
	parent := filepath.ToSlash(filepath.Dir(filepath.FromSlash(old)))
	if parent == "." {
		parent = ""
	}
	newPath := name
	if parent != "" {
		newPath = parent + "/" + name
	}
	if newPath == old {
		m.message = "Folder name unchanged"
		return
	}
	for _, f := range m.folders {
		if (f == newPath || strings.HasPrefix(f, newPath+"/")) && f != old && !strings.HasPrefix(f, old+"/") {
			m.message = "A folder with that name already exists"
			return
		}
	}
	root := filepath.Dir(m.dataPath)
	if err := os.Rename(folderPath(root, old), folderPath(root, newPath)); err != nil {
		m.message = "Could not rename folder: " + err.Error()
		return
	}
	oldFolders := append([]string(nil), m.folders...)
	oldNotes := append([]Note(nil), m.notes...)
	for i, f := range m.folders {
		if f == old {
			m.folders[i] = newPath
		} else if strings.HasPrefix(f, old+"/") {
			m.folders[i] = newPath + strings.TrimPrefix(f, old)
		}
	}
	for i := range m.notes {
		if m.notes[i].Folder == old {
			m.notes[i].Folder = newPath
		} else if strings.HasPrefix(m.notes[i].Folder, old+"/") {
			m.notes[i].Folder = newPath + strings.TrimPrefix(m.notes[i].Folder, old)
		}
	}
	sort.Strings(m.folders)
	if err := atomicJSON(filepath.Join(root, "folders.json"), m.folders); err != nil {
		m.folders = oldFolders
		m.notes = oldNotes
		_ = os.Rename(folderPath(root, newPath), folderPath(root, old))
		m.message = "Could not save renamed folder: " + err.Error()
		return
	}
	if err := persist(m.dataPath, m.notes); err != nil {
		m.folders = oldFolders
		m.notes = oldNotes
		_ = atomicJSON(filepath.Join(root, "folders.json"), oldFolders)
		_ = os.Rename(folderPath(root, newPath), folderPath(root, old))
		m.message = "Could not save note locations: " + err.Error()
		return
	}
	newCollapsed := map[string]bool{}
	for path, collapsed := range m.collapsed {
		if path == old {
			path = newPath
		} else if strings.HasPrefix(path, old+"/") {
			path = newPath + strings.TrimPrefix(path, old)
		}
		newCollapsed[path] = collapsed
	}
	m.collapsed = newCollapsed
	m.message = "Renamed folder to " + newPath
	m.clampCursor()
}

func (m *model) deleteFolder() {
	target := m.pendingFolderDelete
	m.pendingFolderDelete = ""
	m.screen = listScreen
	if target == "" {
		return
	}
	keptNotes := make([]Note, 0, len(m.notes))
	removedNotes := 0
	for _, n := range m.notes {
		if n.Folder == target || strings.HasPrefix(n.Folder, target+"/") {
			removedNotes++
		} else {
			keptNotes = append(keptNotes, n)
		}
	}
	keptFolders := make([]string, 0, len(m.folders))
	for _, f := range m.folders {
		if f != target && !strings.HasPrefix(f, target+"/") {
			keptFolders = append(keptFolders, f)
		}
	}
	root := filepath.Dir(m.dataPath)
	if err := persist(m.dataPath, keptNotes); err != nil {
		m.message = "Could not delete folder contents: " + err.Error()
		return
	}
	if err := atomicJSON(filepath.Join(root, "folders.json"), keptFolders); err != nil {
		m.message = "Could not update folders: " + err.Error()
		return
	}
	m.notes, m.folders = keptNotes, keptFolders
	for path := range m.collapsed {
		if path == target || strings.HasPrefix(path, target+"/") {
			delete(m.collapsed, path)
		}
	}
	if err := os.RemoveAll(folderPath(root, target)); err != nil {
		m.message = "Folder data removed, but its directory could not be cleaned up: " + err.Error()
	} else {
		m.message = fmt.Sprintf("Deleted %s and %d note(s)", target, removedNotes)
	}
	m.clampCursor()
}

func (m *model) placeMovingNote(folder string) {
	for i := range m.notes {
		if m.notes[i].ID == m.movingID {
			m.notes[i].Folder = folder
			m.notes[i].Updated = time.Now()
			break
		}
	}
	if err := persist(m.dataPath, m.notes); err != nil {
		m.message = "Could not move note: " + err.Error()
		return
	}
	m.movingID = ""
	m.message = "Note moved to " + displayFolder(folder)
}

func (m *model) toggleChecked(id string) {
	for i := range m.notes {
		if m.notes[i].ID == id {
			m.notes[i].Checked = !m.notes[i].Checked
			m.notes[i].Updated = time.Now()
			break
		}
	}
	if err := persist(m.dataPath, m.notes); err != nil {
		m.message = "Could not save check state: " + err.Error()
	}
}

func (m *model) removeSelected() {
	e, ok := m.selectedEntry()
	if !ok || e.folder {
		return
	}
	for i := range m.notes {
		if m.notes[i].ID == e.note.ID {
			m.notes = append(m.notes[:i], m.notes[i+1:]...)
			break
		}
	}
	m.screen = listScreen
	m.message = ""
	if err := persist(m.dataPath, m.notes); err != nil {
		m.message = "Could not save deletion: " + err.Error()
	}
	m.clampCursor()
}
