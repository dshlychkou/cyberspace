package tui

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/dshlychkou/cyberspace/internal/game"
)

func (m *Model) loadSaves() {
	saves, err := game.ListSaveFiles(m.cfg.SaveDir)
	if err != nil {
		m.statusMsg = fmt.Sprintf("Failed to list saves: %v", err)
		m.saveFiles = nil
		return
	}
	m.saveFiles = saves
	m.loadIdx = 0
}

func (m *Model) updateLoad(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyPressMsg); ok {
		switch msg.String() {
		case keyCtrlC:
			return m, tea.Quit
		case keyEsc:
			m.screen = screenMenu
		case keyUp, "k":
			if m.loadIdx > 0 {
				m.loadIdx--
			}
		case keyDown, "j":
			if m.loadIdx < len(m.saveFiles)-1 {
				m.loadIdx++
			}
		case keyEnter:
			if len(m.saveFiles) > 0 {
				return m.loadGame(m.saveFiles[m.loadIdx].Path)
			}
		case "d":
			if len(m.saveFiles) > 0 {
				path := m.saveFiles[m.loadIdx].Path
				if err := os.Remove(path); err != nil {
					m.statusMsg = fmt.Sprintf("Failed to delete: %v", err)
				} else {
					m.loadSaves()
					if m.loadIdx >= len(m.saveFiles) && m.loadIdx > 0 {
						m.loadIdx = len(m.saveFiles) - 1
					}
				}
			}
		}
	}
	return m, nil
}

func renderLoadScreen(saves []game.SaveFileInfo, idx, width, height int) string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorNeonPurple).
		Render("L O A D   G A M E")

	var body string
	if len(saves) == 0 {
		body = lipgloss.NewStyle().Foreground(colorDim).Render("No saved games found.")
	} else {
		var lines []string
		for i, s := range saves {
			entry := s.ModTime.Format("2006-01-02  15:04:05")
			if i == idx {
				cursor := lipgloss.NewStyle().Foreground(colorNeonPink).Bold(true).Render("▸ ")
				label := lipgloss.NewStyle().Foreground(colorNeonPink).Bold(true).Render(entry)
				lines = append(lines, cursor+label)
			} else {
				lines = append(lines, "  "+lipgloss.NewStyle().Foreground(colorNeonViolet).Render(entry))
			}
		}
		body = strings.Join(lines, "\n")
	}

	footer := lipgloss.NewStyle().Foreground(colorDim).
		Render("enter: load  d: delete  esc: back")

	panel := stylePanel.Width(54).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			title,
			"",
			body,
			"",
			footer,
		),
	)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, panel)
}
