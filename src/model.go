package main

import (
	"github.com/charmbracelet/bubbles/textinput"
	"time"
)

type Note struct {
	ID      string    `json:"id"`
	Title   string    `json:"title"`
	Body    string    `json:"body"`
	Folder  string    `json:"folder,omitempty"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
	Checked bool      `json:"checked"`
}
type screen int

const (
	listScreen screen = iota
	deleteScreen
)

type treeEntry struct {
	folder   bool
	path     string
	depth    int
	note     Note
	expanded bool
}
type model struct {
	screen              screen
	notes               []Note
	folders             []string
	collapsed           map[string]bool
	cursor              int
	query               textinput.Model
	folderInput         textinput.Model
	creatingFolder      bool
	renamingFolder      bool
	folderParent        string
	folderTarget        string
	pendingFolderDelete string
	editingID           string
	editingFolder       string
	movingID            string
	width, height       int
	message             string
	dataPath            string
}
type editResultMsg struct {
	title, body string
	err         error
}
