package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/dshlychkou/cyberspace/internal/game"
)

var menuItems = []string{"Play", "Settings", "About"}

type settingItem struct {
	Label  string
	Desc   string
	Format func(game.Config) string
	Inc    func(*game.Config)
	Dec    func(*game.Config)
}

func intSetting(label, desc string, get func(game.Config) int, set func(*game.Config, int), step, lo, hi int) settingItem {
	return settingItem{
		Label:  label,
		Desc:   desc,
		Format: func(c game.Config) string { return fmt.Sprintf("%d", get(c)) },
		Inc: func(c *game.Config) {
			v := get(*c) + step
			if v <= hi {
				set(c, v)
			}
		},
		Dec: func(c *game.Config) {
			v := get(*c) - step
			if v >= lo {
				set(c, v)
			}
		},
	}
}

var settingsItems = []settingItem{
	{
		Label:  "Tick Rate",
		Desc:   "Game speed. Lower = faster ticks, more intense gameplay.",
		Format: func(c game.Config) string { return c.TickRate.String() },
		Inc: func(c *game.Config) {
			if c.TickRate < 5*time.Second {
				c.TickRate += 100 * time.Millisecond
			}
		},
		Dec: func(c *game.Config) {
			if c.TickRate > 100*time.Millisecond {
				c.TickRate -= 100 * time.Millisecond
			}
		},
	},
	intSetting("Initial Programs",
		"Programs you start with, clustered on one server.",
		func(c game.Config) int { return c.InitialPrograms }, func(c *game.Config, v int) { c.InitialPrograms = v }, 1, 1, 20),
	intSetting("Initial ICE",
		"Enemy ICE placed on firewalls at game start. More = harder opening.",
		func(c game.Config) int { return c.InitialICE }, func(c *game.Config, v int) { c.InitialICE = v }, 1, 0, 20),
	intSetting("Virus Lifespan",
		"Ticks before a deployed virus expires. Longer = more ICE converted.",
		func(c game.Config) int { return c.VirusLifespan }, func(c *game.Config, v int) { c.VirusLifespan = v }, 1, 1, 50),
	intSetting("Core Win Threshold",
		"Programs needed on CORE to start the win countdown. Higher = harder.",
		func(c game.Config) int { return c.CoreWinThreshold }, func(c *game.Config, v int) { c.CoreWinThreshold = v }, 1, 1, 10),
	intSetting("Core Win Duration",
		"Consecutive ticks holding CORE with enough programs to win.",
		func(c game.Config) int { return c.CoreWinDuration }, func(c *game.Config, v int) { c.CoreWinDuration = v }, 1, 1, 50),
	intSetting("Data Harvest Rate",
		"Data earned per tick per program on a vault. Main income source.",
		func(c game.Config) int { return c.DataHarvestRate }, func(c *game.Config, v int) { c.DataHarvestRate = v }, 1, 1, 50),
	intSetting("Program Spawn Cost",
		"Data spent to manually spawn a program (S key). Higher = slower expansion.",
		func(c game.Config) int { return c.ProgramSpawnCost }, func(c *game.Config, v int) { c.ProgramSpawnCost = v }, 5, 0, 200),
	intSetting("Virus Deploy Cost",
		"Compute spent to deploy a virus (V key). Viruses convert nearby ICE.",
		func(c game.Config) int { return c.VirusDeployCost }, func(c *game.Config, v int) { c.VirusDeployCost = v }, 5, 0, 200),
	intSetting("Program Upkeep",
		"Data drained per program per tick. More programs = higher burn rate.",
		func(c game.Config) int { return c.ProgramUpkeep }, func(c *game.Config, v int) { c.ProgramUpkeep = v }, 1, 0, 10),
	intSetting("Core Hold Cost",
		"Compute drained per program on CORE per tick. Holding core is expensive.",
		func(c game.Config) int { return c.CoreHoldCost }, func(c *game.Config, v int) { c.CoreHoldCost = v }, 1, 0, 20),
	intSetting("Survive Min",
		"Min neighbor support for program survival. 0 = programs never die alone.",
		func(c game.Config) int { return c.SurviveMin }, func(c *game.Config, v int) { c.SurviveMin = v }, 1, 0, 10),
	intSetting("Survive Max",
		"Max support before overcrowding kills programs. Prevents deathballs.",
		func(c game.Config) int { return c.SurviveMax }, func(c *game.Config, v int) { c.SurviveMax = v }, 1, 1, 20),
	intSetting("Spread Exact",
		"Neighbor programs needed for auto-spread. Higher = slower free expansion.",
		func(c game.Config) int { return c.SpreadExact }, func(c *game.Config, v int) { c.SpreadExact = v }, 1, 1, 10),
	intSetting("Initial Data",
		"Starting Data resource. Used for spawning programs and paying upkeep.",
		func(c game.Config) int { return c.InitialData }, func(c *game.Config, v int) { c.InitialData = v }, 10, 0, 1000),
	intSetting("Initial Compute",
		"Starting Compute resource. Used for viruses and holding CORE.",
		func(c game.Config) int { return c.InitialCompute }, func(c *game.Config, v int) { c.InitialCompute = v }, 10, 0, 1000),
	intSetting("ICE Spawn Tick",
		"Tick when first new ICE spawns. Earlier = more pressure sooner.",
		func(c game.Config) int { return c.ICESpawnTick }, func(c *game.Config, v int) { c.ICESpawnTick = v }, 5, 5, 200),
	intSetting("ICE Escalation Tick",
		"Tick when ICE bursts begin (all firewalls + core). The difficulty spike.",
		func(c game.Config) int { return c.ICEEscalationTick }, func(c *game.Config, v int) { c.ICEEscalationTick = v }, 5, 10, 500),
}

func renderMenu(idx, width, height int) string {
	// ASCII art title with neon purple glow
	logoLines := []string{
		" ██████╗██╗   ██╗██████╗ ███████╗██████╗ ",
		"██╔════╝╚██╗ ██╔╝██╔══██╗██╔════╝██╔══██╗",
		"██║      ╚████╔╝ ██████╔╝█████╗  ██████╔╝",
		"██║       ╚██╔╝  ██╔══██╗██╔══╝  ██╔══██╗",
		"╚██████╗   ██║   ██████╔╝███████╗██║  ██║",
		" ╚═════╝   ╚═╝   ╚═════╝ ╚══════╝╚═╝  ╚═╝",
	}

	titleStyle := lipgloss.NewStyle().Foreground(colorNeonPurple).Bold(true)
	var logo []string
	for _, line := range logoLines {
		logo = append(logo, titleStyle.Render(line))
	}
	title := strings.Join(logo, "\n")

	subtitle := lipgloss.NewStyle().Foreground(colorNeonPink).Render("   S   P   A   C   E")

	var items []string
	for i, item := range menuItems {
		if i == idx {
			cursor := lipgloss.NewStyle().Foreground(colorNeonPink).Bold(true).Render("▸ ")
			label := lipgloss.NewStyle().Foreground(colorNeonPink).Bold(true).Render(item)
			items = append(items, cursor+label)
		} else {
			items = append(items, "  "+lipgloss.NewStyle().Foreground(colorNeonViolet).Render(item))
		}
	}

	divider := lipgloss.NewStyle().Foreground(colorNeonViolet).Render("────────────────────")

	menu := lipgloss.JoinVertical(lipgloss.Center,
		title,
		subtitle,
		"",
		divider,
		"",
		strings.Join(items, "\n"),
		"",
		divider,
		"",
		lipgloss.NewStyle().Foreground(colorDim).Render("arrows: navigate  enter: select  q: quit"),
	)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, menu)
}

func renderSettings(cfg *game.Config, selectedIdx, width, height int) string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorNeonPurple).
		Render("S E T T I N G S")

	labelStyle := lipgloss.NewStyle().Foreground(colorWhite).Width(22)
	valueStyle := lipgloss.NewStyle().Foreground(colorNeonCyan)
	selectedLabel := lipgloss.NewStyle().Foreground(colorNeonPink).Bold(true).Width(22)
	selectedValue := lipgloss.NewStyle().Foreground(colorNeonPink).Bold(true)
	cursorStyle := lipgloss.NewStyle().Foreground(colorNeonPink).Bold(true)
	arrowStyle := lipgloss.NewStyle().Foreground(colorNeonPurple).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(colorNeonViolet).Italic(true)

	var rows []string
	for i, item := range settingsItems {
		val := item.Format(*cfg)
		if i == selectedIdx {
			rows = append(rows,
				cursorStyle.Render("> ")+selectedLabel.Render(item.Label)+arrowStyle.Render("< ")+selectedValue.Render(val)+arrowStyle.Render(" >"),
				"    "+descStyle.Render(item.Desc),
			)
		} else {
			rows = append(rows, "  "+labelStyle.Render(item.Label)+"  "+valueStyle.Render(val))
		}
	}

	body := strings.Join(rows, "\n")

	footer := lipgloss.NewStyle().Foreground(colorNeonViolet).
		Render("arrows: navigate  left/right: adjust  esc: back")

	panel := stylePanel.Width(62).Render(
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

func renderAbout(width, height int) string {
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(colorNeonPurple).
		Render("A B O U T")

	desc := lipgloss.NewStyle().Foreground(colorWhite).Render(
		"CYBERSPACE is a terminal network strategy game.\n" +
			"Infiltrate a cyberpunk network, deploy programs,\n" +
			"hack through ICE defenses, spread viruses, and\n" +
			"capture the CORE to win.")

	controls := lipgloss.NewStyle().Foreground(colorNeonPink).Bold(true).Render("Controls") + "\n" +
		lipgloss.NewStyle().Foreground(colorWhite).Render(
			"  Space      Toggle pause\n"+
				"  arrows     Select node\n"+
				"  s          Spawn program (costs Data)\n"+
				"  v          Deploy virus (costs Compute)\n"+
				"  +/-        Adjust speed\n"+
				"  q          Quit")

	economy := lipgloss.NewStyle().Foreground(colorNeonPink).Bold(true).Render("Economy") + "\n" +
		lipgloss.NewStyle().Foreground(colorWhite).Render(
			"  Every program costs Data each tick (upkeep).\n"+
				"  Programs on Vaults earn Data (income).\n"+
				"  Programs on Relays earn Compute.\n"+
				"  Holding CORE drains Compute per program.\n"+
				"  If Data hits 0, programs starve and die.\n"+
				"  If Compute hits 0, CORE programs fail.\n"+
				"  Balance expansion vs income to survive!")

	legend := lipgloss.NewStyle().Foreground(colorNeonPink).Bold(true).Render("Map Symbols") + "\n" +
		styleCore.Render("  ★") + " Core (target)   " + styleProgram.Render("P") + " Program\n" +
		styleFirewall.Render("  ◆") + " Firewall        " + styleICE.Render("I") + " ICE (enemy)\n" +
		styleProgram.Render("  ◆") + " Server          " + styleVirus.Render("V") + " Virus\n" +
		styleEvent.Render("  ◇") + " Relay           " +
		styleData.Render("$") + " Data flow\n" +
		styleData.Render("  ◆") + " Vault           " +
		styleProgram.Render("~") + " Compute flow\n" +
		"                    " + styleICE.Render("×") + " ICE threat"

	footer := lipgloss.NewStyle().Foreground(colorNeonViolet).Render("Press Esc to return.")

	panel := stylePanel.Width(54).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			title,
			"",
			desc,
			"",
			controls,
			"",
			economy,
			"",
			legend,
			"",
			footer,
		),
	)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, panel)
}
