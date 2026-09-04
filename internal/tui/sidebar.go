package tui

import (
	"fmt"
	"strings"

	"github.com/dshlychkou/cyberspace/internal/game"
)

func renderSidebar(snap *game.StateSnapshot, _ int) string {
	var sb strings.Builder
	sidebarObjective(&sb, snap)
	sidebarResources(&sb, snap)
	sidebarControls(&sb, snap)
	sidebarNodeTypes(&sb, snap)
	sidebarEntities(&sb)
	sidebarRules(&sb)
	sidebarEconomy(&sb, snap)
	return sb.String()
}

func sidebarObjective(sb *strings.Builder, snap *game.StateSnapshot) {
	sb.WriteString(styleTitle.Render("OBJECTIVE"))
	sb.WriteByte('\n')
	fmt.Fprintf(sb, "Get %d+ ", snap.CoreWinThreshold)
	sb.WriteString(styleProgram.Render("Programs"))
	sb.WriteString(" to\n")
	sb.WriteString(styleCore.Render("★ CORE"))
	fmt.Fprintf(sb, " and hold %d ticks.\n", snap.CoreWinDuration)
	sb.WriteByte('\n')
	sb.WriteString(styleTitle.Render("TIP"))
	sb.WriteByte('\n')
	sb.WriteString("Hold ")
	sb.WriteString(styleData.Render("Vaults"))
	sb.WriteString(" for Data income\n")
	sb.WriteString("or your programs starve!\n")
	sb.WriteByte('\n')
}

func sidebarResources(sb *strings.Builder, snap *game.StateSnapshot) {
	sb.WriteString(styleTitle.Render("RESOURCES"))
	sb.WriteByte('\n')
	fmt.Fprintf(sb, "Data:    %s\n", styleData.Render(fmt.Sprintf("%d", snap.Resources.Data)))
	fmt.Fprintf(sb, "Compute: %s\n", styleData.Render(fmt.Sprintf("%d", snap.Resources.Compute)))
	fmt.Fprintf(sb, "Score:   %s\n", styleScore.Render(fmt.Sprintf("%d", snap.Score)))
	sb.WriteByte('\n')
}

func sidebarControls(sb *strings.Builder, snap *game.StateSnapshot) {
	sb.WriteString(styleTitle.Render("CONTROLS"))
	sb.WriteByte('\n')
	sb.WriteString(styleSelected.Render("←↑↓→") + " Navigate graph\n")
	sb.WriteString(styleSelected.Render("Click") + " Select node\n")
	sb.WriteString(styleSelected.Render("S") + fmt.Sprintf("   Spawn program\n       costs %d Data\n", snap.ProgramSpawnCost))
	sb.WriteString(styleSelected.Render("V") + fmt.Sprintf("   Deploy virus\n       costs %d Compute\n", snap.VirusDeployCost))
	sb.WriteString(styleSelected.Render("Spc") + " Pause / Resume\n")
	sb.WriteString(styleSelected.Render("+/-") + " Speed up / down\n")
	sb.WriteString(styleSelected.Render("Esc") + " Main menu\n")
	sb.WriteByte('\n')
}

func sidebarNodeTypes(sb *strings.Builder, snap *game.StateSnapshot) {
	sb.WriteString(styleTitle.Render("NODES"))
	sb.WriteByte('\n')
	sb.WriteString(styleProgram.Render("◆S"))
	sb.WriteString("rv  Auto-spread hub\n")
	sb.WriteString(styleData.Render("◆V"))
	fmt.Fprintf(sb, "lt  +%d Data/prog/tick\n", snap.DataHarvestRate)
	sb.WriteString(styleEvent.Render("◇R"))
	fmt.Fprintf(sb, "ly  +%d Compute/prog/tick\n", snap.ComputeHarvestRate)
	sb.WriteString(styleFirewall.Render("◆F"))
	sb.WriteString("W   Blocks spread, ICE\n")
	sb.WriteString(styleCore.Render("★C"))
	sb.WriteString("ORE Target, hold to win\n")
	sb.WriteByte('\n')
}

func sidebarEntities(sb *strings.Builder) {
	sb.WriteString(styleTitle.Render("ENTITIES"))
	sb.WriteByte('\n')
	sb.WriteString(styleProgram.Render("P"))
	sb.WriteString(" Program (yours)\n")
	sb.WriteString(styleICE.Render("I"))
	sb.WriteString(" ICE (enemy defense)\n")
	sb.WriteString(styleVirus.Render("V"))
	sb.WriteString(" Virus (converts ICE)\n")
	sb.WriteByte('\n')
}

func sidebarRules(sb *strings.Builder) {
	sb.WriteString(styleTitle.Render("RULES"))
	sb.WriteByte('\n')
	sb.WriteString("Auto-spread: 3+ neighbor\n")
	sb.WriteString("programs on srv/rly/vlt.\n")
	sb.WriteString("FW and CORE: manual ")
	sb.WriteString(styleSelected.Render("S"))
	sb.WriteString(".\n")
	sb.WriteString("ICE>prog → prog dies.\n")
	sb.WriteString("Virus flips nearby ICE.\n")
	sb.WriteByte('\n')
}

func sidebarEconomy(sb *strings.Builder, snap *game.StateSnapshot) {
	sb.WriteString(styleTitle.Render("ECONOMY"))
	sb.WriteByte('\n')
	sb.WriteString(styleScore.Render("+"))
	fmt.Fprintf(sb, " Vault: +%d Data/prog\n", snap.DataHarvestRate)
	sb.WriteString(styleScore.Render("+"))
	fmt.Fprintf(sb, " Relay: +%d Compute/prog\n", snap.ComputeHarvestRate)
	sb.WriteString(styleError.Render("-"))
	fmt.Fprintf(sb, " Upkeep: -%d Data/prog\n", snap.ProgramUpkeep)
	sb.WriteString(styleError.Render("-"))
	fmt.Fprintf(sb, " CORE:   -%d Compute/prog\n", snap.CoreHoldCost)
	sb.WriteString(styleEvent.Render("Bankrupt = death!\n"))
}

func countEntities(n game.NodeSnapshot, snap *game.StateSnapshot) (programs, ices, viruses int) {
	programByID := make(map[int]bool, len(snap.Programs))
	for _, p := range snap.Programs {
		programByID[p.ID] = true
	}
	iceByID := make(map[int]bool, len(snap.ICEs))
	for _, ice := range snap.ICEs {
		iceByID[ice.ID] = true
	}
	virusByID := make(map[int]bool, len(snap.Viruses))
	for _, v := range snap.Viruses {
		virusByID[v.ID] = true
	}
	for _, eid := range n.Entities {
		switch {
		case programByID[eid]:
			programs++
		case iceByID[eid]:
			ices++
		case virusByID[eid]:
			viruses++
		}
	}
	return
}
