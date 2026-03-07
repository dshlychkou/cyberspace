package tui

import (
	"fmt"
	"strings"

	"github.com/dshlychkou/cyberspace/internal/game"
)

func renderSidebar(snap game.StateSnapshot, width int) string {
	var sb strings.Builder

	// Goal
	sb.WriteString(styleTitle.Render("OBJECTIVE"))
	sb.WriteByte('\n')
	sb.WriteString(fmt.Sprintf("Get %d+ ", snap.CoreWinThreshold))
	sb.WriteString(styleProgram.Render("Programs"))
	sb.WriteString(" to\n")
	sb.WriteString(styleCore.Render("★ CORE"))
	sb.WriteString(fmt.Sprintf(" and hold %d ticks.\n", snap.CoreWinDuration))
	sb.WriteByte('\n')

	// Resources
	sb.WriteString(styleTitle.Render("RESOURCES"))
	sb.WriteByte('\n')
	sb.WriteString(fmt.Sprintf("Data:    %s\n", styleData.Render(fmt.Sprintf("%d", snap.Resources.Data))))
	sb.WriteString(fmt.Sprintf("Compute: %s\n", styleData.Render(fmt.Sprintf("%d", snap.Resources.Compute))))
	sb.WriteString(fmt.Sprintf("Score:   %s\n", styleScore.Render(fmt.Sprintf("%d", snap.Score))))
	sb.WriteByte('\n')

	// Controls
	sb.WriteString(styleTitle.Render("CONTROLS"))
	sb.WriteByte('\n')
	sb.WriteString(styleSelected.Render("↑/↓") + " Select node\n")
	sb.WriteString(styleSelected.Render("S") + fmt.Sprintf("   Spawn program\n       costs %d Data\n", snap.ProgramSpawnCost))
	sb.WriteString(styleSelected.Render("V") + fmt.Sprintf("   Deploy virus\n       costs %d Compute\n", snap.VirusDeployCost))
	sb.WriteString(styleSelected.Render("Spc") + " Pause / Resume\n")
	sb.WriteString(styleSelected.Render("+/-") + " Speed up / down\n")
	sb.WriteString(styleSelected.Render("Q") + "   Quit\n")
	sb.WriteByte('\n')

	// Rules explained in plain language
	sb.WriteString(styleTitle.Render("HOW IT WORKS"))
	sb.WriteByte('\n')
	sb.WriteString("Programs spread auto-\n")
	sb.WriteString("matically between servers,\n")
	sb.WriteString("relays, and vaults.\n")
	sb.WriteByte('\n')
	sb.WriteString("Firewalls and Core\n")
	sb.WriteString(styleFirewall.Render("block"))
	sb.WriteString(" auto-spread.\n")
	sb.WriteString("You must place programs\n")
	sb.WriteString("there manually with ")
	sb.WriteString(styleSelected.Render("S"))
	sb.WriteString(".\n")
	sb.WriteByte('\n')
	sb.WriteString("If a node has more ICE\n")
	sb.WriteString("than programs, the ICE\n")
	sb.WriteString("destroys all programs\n")
	sb.WriteString("on that node.\n")
	sb.WriteByte('\n')
	sb.WriteString("A virus converts one\n")
	sb.WriteString("nearby ICE into a\n")
	sb.WriteString("program each tick.\n")
	sb.WriteByte('\n')

	// Economy
	sb.WriteString(styleTitle.Render("EARNING RESOURCES"))
	sb.WriteByte('\n')
	sb.WriteString("Program on a vault:\n")
	sb.WriteString("  +5 Data per tick\n")
	sb.WriteString("Program on a relay:\n")
	sb.WriteString("  +2 Compute per tick\n")

	return sb.String()
}

func countEntities(n game.NodeSnapshot, snap game.StateSnapshot) (programs, ices, viruses int) {
	for _, eid := range n.Entities {
		for _, p := range snap.Programs {
			if p.ID == eid {
				programs++
			}
		}
		for _, ice := range snap.ICEs {
			if ice.ID == eid {
				ices++
			}
		}
		for _, v := range snap.Viruses {
			if v.ID == eid {
				viruses++
			}
		}
	}
	return
}
