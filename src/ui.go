package main

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"path/filepath"
	"strings"
	"time"
)

var (
	muted         = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	accent        = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	titleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Bold(true)
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("24")).Bold(true).Padding(0, 1)
	panelStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("238")).Padding(0, 1)
)

func (m model) View() string {
	width, height := m.width, m.height
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	return m.dashboardView(width, height)
}

func (m model) dashboardView(width, height int) string {
	innerW, innerH := max(1, width-4), max(1, height-2)
	done := 0
	for _, n := range m.notes {
		if n.Checked {
			done++
		}
	}
	controls := "j/k move · enter open/fold · h/l fold · n note · f folder · r rename · d delete · x check · m/p move · / search · q quit"
	if innerW < 130 {
		controls = "↑/↓ move · enter open/fold · n note · f folder · r rename · d delete · / search · q quit"
	}
	if innerW < 80 {
		controls = "j/k move · enter open · n note · f folder · d delete · q quit"
	}
	if innerW < 54 {
		controls = "j/k · enter · n · f · d · q"
	}
	controlBar := lipgloss.NewStyle().Foreground(lipgloss.Color("117")).Background(lipgloss.Color("235")).Padding(0, 1).Render(truncateCells(controls, max(1, innerW-2)))
	top := lipgloss.JoinVertical(lipgloss.Left, m.headerView(width, done), controlBar)
	if m.query.Focused() {
		q := m.query
		q.Width = max(8, innerW-2)
		top = lipgloss.JoinVertical(lipgloss.Left, top, q.View())
	} else if strings.TrimSpace(m.query.Value()) != "" {
		top = lipgloss.JoinVertical(lipgloss.Left, top, muted.Render("Search: "+truncateCells(m.query.Value(), innerW)))
	}
	if m.creatingFolder || m.renamingFolder {
		fi := m.folderInput
		fi.Width = max(8, innerW-2)
		label := "New folder in " + displayFolder(m.folderParent)
		prompt := "enter create · esc cancel"
		if m.renamingFolder {
			label = "Rename folder " + m.folderTarget
			prompt = "enter rename · esc cancel"
		}
		top = lipgloss.JoinVertical(lipgloss.Left, top, titleStyle.Render(truncateCells(label, innerW)), fi.View()+"  "+muted.Render(prompt))
	} else if m.screen == deleteScreen {
		var title string
		if m.pendingFolderDelete != "" {
			noteCount, folderCount := 0, 0
			for _, n := range m.notes {
				if n.Folder == m.pendingFolderDelete || strings.HasPrefix(n.Folder, m.pendingFolderDelete+"/") {
					noteCount++
				}
			}
			for _, f := range m.folders {
				if strings.HasPrefix(f, m.pendingFolderDelete+"/") {
					folderCount++
				}
			}
			title = fmt.Sprintf("Delete %s/ plus %d folders and %d notes?", m.pendingFolderDelete, folderCount, noteCount)
		} else if e, ok := m.selectedEntry(); ok && !e.folder {
			title = fmt.Sprintf("Delete note %q?", e.note.Title)
		}
		alertStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("52")).Bold(true).Padding(0, 1)
		alertText := lipgloss.JoinVertical(lipgloss.Left, truncateCells(title, max(1, innerW-2)), "Enter/y confirm · n/Esc cancel")
		alert := alertStyle.Width(max(1, innerW-2)).Render(alertText)
		top = lipgloss.JoinVertical(lipgloss.Left, top, alert)
	} else if m.message != "" {
		status := lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("24")).Padding(0, 1).Render(truncateCells(m.message, max(1, innerW-2)))
		top = lipgloss.JoinVertical(lipgloss.Left, top, status)
	}
	topH := lipgloss.Height(top)
	panelH := max(1, innerH-topH-1)
	rows := m.treeRows()
	cursor := m.cursor
	if cursor >= len(rows) {
		cursor = len(rows) - 1
	}
	if cursor < 0 {
		cursor = 0
	}
	var main string
	if innerW >= 92 && panelH >= 7 {
		treeW := innerW * 56 / 100
		if innerW-treeW < 30 {
			treeW = innerW - 30
		}
		detailW := innerW - treeW
		main = lipgloss.JoinHorizontal(lipgloss.Top, m.treePanel(rows, cursor, treeW, panelH), m.detailPanel(treeW, detailW, panelH))
	} else if innerW >= 70 && panelH >= 18 {
		treeH := panelH * 57 / 100
		if treeH < 8 {
			treeH = 8
		}
		if panelH-treeH-1 < 7 {
			treeH = panelH - 8
		}
		detailH := max(1, panelH-treeH-1)
		main = lipgloss.JoinVertical(lipgloss.Left, m.treePanel(rows, cursor, innerW, treeH), m.detailPanel(0, innerW, detailH))
	} else {
		main = m.treePanel(rows, cursor, innerW, panelH)
	}
	fill := max(0, innerH-topH-lipgloss.Height(main)-1)
	content := top + "\n" + main + strings.Repeat("\n", fill)
	return lipgloss.NewStyle().Width(innerW).Height(innerH).Padding(1, 2).MaxWidth(width).MaxHeight(height).Render(content)
}

func (m model) headerView(width, done int) string {
	date := time.Now().Format("Mon 02 Jan 2006 · 15:04")
	stats := fmt.Sprintf("%d notes · %d checked", len(m.notes), done)
	if width >= 98 && m.height >= 22 {
		art := accent.Render("██████╗ ██╗   ██╗██╗") + "\n" + accent.Render("██╔══██╗██║   ██║██║") + "\n" + accent.Render("██████╔╝██║   ██║██║") + "\n" + accent.Render("██╔═══╝ ██║   ██║██║") + "\n" + accent.Render("██║     ╚██████╔╝██║") + "\n" + accent.Render("╚═╝      ╚═════╝ ╚═╝")
		info := lipgloss.NewStyle().Padding(1, 0).Render(titleStyle.Render("YOUR THOUGHTS, IN ORDER") + "\n" + muted.Render(date) + "\n\n" + accent.Render(stats))
		return lipgloss.JoinHorizontal(lipgloss.Top, art, "   ", info)
	}
	line := accent.Render("PUI") + muted.Render("  ·  "+stats)
	if width >= 70 {
		line += "  ·  " + muted.Render(date)
	}
	return truncateStyled(line, max(8, width-4))
}

func (m model) treePanel(rows []treeEntry, cursor, width, height int) string {
	insideW := max(1, width-4)
	rowLimit := max(1, height-5)
	start := 0
	if cursor >= rowLimit {
		start = cursor - rowLimit + 1
	}
	if start < 0 {
		start = 0
	}
	end := min(len(rows), start+rowLimit)
	lines := []string{titleStyle.Render("FILES")}
	if len(rows) == 0 {
		if strings.TrimSpace(m.query.Value()) != "" {
			lines = append(lines, muted.Render("No matches."))
		} else {
			lines = append(lines, muted.Render("Empty. Press n for a note or f for a folder."))
		}
	} else {
		for i := start; i < end; i++ {
			e := rows[i]
			indent := strings.Repeat("  ", e.depth)
			var line string
			if e.folder {
				marker := "▾"
				if !e.expanded {
					marker = "▸"
				}
				label := filepath.Base(e.path) + "/"
				if e.path == "" {
					label = "Root"
				}
				line = indent + marker + " " + label
			} else {
				check := "[ ]"
				if e.note.Checked {
					check = "[✓]"
				}
				prefix := indent + "  " + check + " "
				if e.note.ID == m.movingID {
					prefix = indent + "✂ " + check + " "
				}
				line = prefix + truncateCells(e.note.Title, max(1, insideW-lipgloss.Width(prefix)-3))
			}
			if i == cursor {
				lines = append(lines, selectedStyle.Width(max(1, insideW-2)).Render("▸ "+truncateCells(line, max(1, insideW-4))))
			} else {
				lines = append(lines, "  "+truncateCells(line, insideW-2))
			}
		}
	}
	return panelStyle.Width(max(1, width-2)).Height(max(1, height-2)).Render(strings.Join(lines, "\n"))
}

func (m model) detailPanel(treeWidth, width, height int) string {
	var body string
	if e, ok := m.selectedEntry(); ok {
		if e.folder {
			label := e.path + "/"
			if e.path == "" {
				label = "Root"
			}
			body = titleStyle.Render(label) + "\n" + muted.Render("FOLDER")
		} else {
			status := "○ OPEN"
			if e.note.Checked {
				status = accent.Render("✓ CHECKED")
			}
			body = titleStyle.Render(truncateCells(e.note.Title, max(8, width-8))) + "\n" + status + "\n" + muted.Render("IN  "+displayFolder(e.note.Folder)) + "\n" + muted.Render("CREATED  "+e.note.Created.Format("02 Jan 2006 · 15:04")) + "\n" + muted.Render("UPDATED  "+e.note.Updated.Format("02 Jan 2006 · 15:04"))
			maxLines := max(1, height-11)
			if width < 80 {
				maxLines = max(1, height-10)
			}
			preview := previewLines(strings.TrimSpace(e.note.Body), max(8, width-8), maxLines)
			if preview != "" {
				body += "\n\n" + preview
			}
		}
	} else {
		body = muted.Render("Select a note to preview it.")
	}
	return panelStyle.Width(max(1, width-2)).Height(max(1, height-2)).Render(body)
}

func previewLines(text string, width, lines int) string {
	if text == "" {
		return muted.Render("No note text.")
	}
	parts := strings.Split(text, "\n")
	shown := min(len(parts), lines)
	for i := 0; i < shown; i++ {
		parts[i] = truncateCells(parts[i], width)
	}
	if len(parts) > shown && shown > 0 {
		parts[shown-1] = truncateCells(parts[shown-1], max(1, width-1)) + "…"
	}
	return strings.Join(parts[:shown], "\n")
}

func truncateStyled(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}
	return lipgloss.NewStyle().MaxWidth(width).Render(s)
}

func truncateCells(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	var b strings.Builder
	used := 0
	for _, r := range s {
		rw := lipgloss.Width(string(r))
		if used+rw > max(0, width-1) {
			break
		}
		b.WriteRune(r)
		used += rw
	}
	b.WriteRune('…')
	return b.String()
}
