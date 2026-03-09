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
	fmt.Fprintf(&sb, "Get %d+ ", snap.CoreWinThreshold)
	sb.WriteString(styleProgram.Render("Programs"))
	sb.WriteString(" to\n")
	sb.WriteString(styleCore.Render("★ CORE"))
	fmt.Fprintf(&sb, " and hold %d ticks.\n", snap.CoreWinDuration)
	sb.WriteByte('\n')

	// Resources
	sb.WriteString(styleTitle.Render("RESOURCES"))
	sb.WriteByte('\n')
	fmt.Fprintf(&sb, "Data:    %s\n", styleData.Render(fmt.Sprintf("%d", snap.Resources.Data)))
	fmt.Fprintf(&sb, "Compute: %s\n", styleData.Render(fmt.Sprintf("%d", snap.Resources.Compute)))
	fmt.Fprintf(&sb, "Score:   %s\n", styleScore.Render(fmt.Sprintf("%d", snap.Score)))
	sb.WriteByte('\n')

	// Controls
	sb.WriteString(styleTitle.Render("CONTROLS"))
	sb.WriteByte('\n')
	sb.WriteString(styleSelected.Render("←↑↓→") + " Navigate graph\n")
	sb.WriteString(styleSelected.Render("Click") + " Select node\n")
	sb.WriteString(styleSelected.Render("S") + fmt.Sprintf("   Spawn program\n       costs %d Data\n", snap.ProgramSpawnCost))
	sb.WriteString(styleSelected.Render("V") + fmt.Sprintf("   Deploy virus\n       costs %d Compute\n", snap.VirusDeployCost))
	sb.WriteString(styleSelected.Render("Spc") + " Pause / Resume\n")
	sb.WriteString(styleSelected.Render("+/-") + " Speed up / down\n")
	sb.WriteString(styleSelected.Render("Esc") + " Main menu\n")
	sb.WriteString(styleSelected.Render("Q") + "   Quit\n")
	sb.WriteByte('\n')

	// Node types
	sb.WriteString(styleTitle.Render("NODES"))
	sb.WriteByte('\n')
	sb.WriteString(styleProgram.Render("◆S"))
	sb.WriteString("rv  Auto-spread hub\n")
	sb.WriteString(styleData.Render("◆V"))
	sb.WriteString("lt  +5 Data/prog/tick\n")
	sb.WriteString(styleEvent.Render("◇R"))
	sb.WriteString("ly  +2 Compute/prog/tick\n")
	sb.WriteString(styleFirewall.Render("◆F"))
	sb.WriteString("W   Blocks spread, ICE\n")
	sb.WriteString(styleCore.Render("★C"))
	sb.WriteString("ORE Target, hold to win\n")
	sb.WriteByte('\n')

	// Entities
	sb.WriteString(styleTitle.Render("ENTITIES"))
	sb.WriteByte('\n')
	sb.WriteString(styleProgram.Render("P"))
	sb.WriteString(" Program (yours)\n")
	sb.WriteString(styleICE.Render("I"))
	sb.WriteString(" ICE (enemy defense)\n")
	sb.WriteString(styleVirus.Render("V"))
	sb.WriteString(" Virus (converts ICE)\n")
	sb.WriteByte('\n')

	// Rules
	sb.WriteString(styleTitle.Render("RULES"))
	sb.WriteByte('\n')
	sb.WriteString("Auto-spread: 3+ neighbor\n")
	sb.WriteString("programs on srv/rly/vlt.\n")
	sb.WriteString("FW and CORE: manual ")
	sb.WriteString(styleSelected.Render("S"))
	sb.WriteString(".\n")
	sb.WriteString("ICE>=prog → prog dies.\n")
	sb.WriteString("Virus flips nearby ICE.\n")
	sb.WriteByte('\n')

	// Economy
	sb.WriteString(styleTitle.Render("ECONOMY"))
	sb.WriteByte('\n')
	sb.WriteString(styleScore.Render("+"))
	sb.WriteString(" Vault: +5 Data/prog\n")
	sb.WriteString(styleScore.Render("+"))
	sb.WriteString(" Relay: +2 Compute/prog\n")
	sb.WriteString(styleError.Render("-"))
	sb.WriteString(" Upkeep: -1 Data/prog\n")
	sb.WriteString(styleError.Render("-"))
	sb.WriteString(" CORE:   -3 Compute/prog\n")
	sb.WriteString(styleEvent.Render("Bankrupt = death!\n"))

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
