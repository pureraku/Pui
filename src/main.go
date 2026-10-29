package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	m, err := newModel()
	if err != nil {
		fmt.Fprintln(os.Stderr, "pui:", err)
		os.Exit(1)
	}
	if _, err = tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "pui:", err)
		os.Exit(1)
	}
}
