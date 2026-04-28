package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the UI based on the current state.
func (m Model) View() string {
	// Skip the first paint until WindowSizeMsg arrives. Rendering at an
	// unknown geometry causes Bubble Tea's renderer to reconcile against the
	// real terminal on the next frame, which can bleed the prior frame down
	// into the rows below the live view (see docs/sidebar-duplication-bug.md).
	if m.width == 0 {
		return ""
	}
	width := m.width
	if width < 80 {
		width = 80
	}

	header := m.renderHeader(width)
	body := m.renderBody(width)
	footer := m.renderFooter(width)

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

// renderHeader: logo on left, status block in middle, versions on right.
func (m Model) renderHeader(width int) string {
	// Left column: logo
	logoBlock := logoStyle.Render(Logo)

	// Middle column: status pill, user@host. Blank leading line for vertical centering vs. logo.
	pill := m.renderStatusPill()
	statusLine := headerLabelStyle.Render("Status: ") + pill + m.renderStatusHint()
	userLine := headerTextStyle.Render(m.host.Username + "@" + m.host.Hostname)
	midRows := []string{"", statusLine}
	if other := m.renderOtherVPNs(); other != "" {
		midRows = append(midRows, other)
	}
	midRows = append(midRows, userLine)
	midContent := lipgloss.JoinVertical(lipgloss.Left, midRows...)

	// Right column: app version.
	rightContent := headerLabelStyle.Render("yabinui:  ") + headerTextStyle.Render(AppVersion)
	rightBlock := lipgloss.JoinVertical(lipgloss.Left, "", rightContent)

	// Three-column row with flexible middle.
	leftW := lipgloss.Width(logoBlock)
	rightW := lipgloss.Width(rightBlock)
	midWidth := width - leftW - rightW - 4 // 4 for outer padding
	if midWidth < 10 {
		midWidth = 10
	}
	midBlock := lipgloss.NewStyle().Width(midWidth).PaddingLeft(2).Render(midContent)

	rightAligned := lipgloss.NewStyle().
		Align(lipgloss.Right).
		Width(rightW + 2).
		PaddingRight(1).
		Render(rightBlock)

	leftPadded := lipgloss.NewStyle().PaddingLeft(1).Render(logoBlock)

	row := lipgloss.JoinHorizontal(lipgloss.Top, leftPadded, midBlock, rightAligned)
	return row + "\n"
}

func (m Model) renderStatusHint() string {
	var hint string
	switch m.state {
	case StateConnected:
		hint = " (press . to disconnect)"
	case StateDisconnected:
		hint = " (press . to connect)"
	case StateError:
		hint = " (press enter to retry)"
	default:
		return ""
	}
	return headerMutedStyle.Render(hint)
}

// renderOtherVPNs returns a header line listing non-Driveline VPNs that are
// currently up, or "" when none are. Keeps the header unchanged in the common
// case where only Driveline (or nothing) is active.
func (m Model) renderOtherVPNs() string {
	dot := lipgloss.NewStyle().Foreground(Success).Render("●")
	var parts []string
	for _, e := range m.wgEntries {
		if e.State == WGUp {
			parts = append(parts, dot+" "+headerTextStyle.Render(e.Name)+" "+headerMutedStyle.Render("(wg)"))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return headerLabelStyle.Render("Other:  ") + strings.Join(parts, "  ")
}

func (m Model) renderStatusPill() string {
	switch m.state {
	case StateConnected:
		return pillConnected.Render("Connected")
	case StateDisconnected:
		return pillDisconnected.Render("Disconnected")
	case StateChecking:
		return pillNeutral.Render(m.spinner.View() + " Checking")
	case StateConnecting:
		return pillNeutral.Render(m.spinner.View() + " Connecting")
	case StateDisconnecting:
		return pillNeutral.Render(m.spinner.View() + " Disconnecting")
	case StateError:
		return pillDisconnected.Render("Error")
	}
	return ""
}

// renderBody: sidebar of tabs on left, tab content on right.
func (m Model) renderBody(width int) string {
	sidebar := m.renderSidebar()
	content := m.renderTabContent(width - SidebarWidth - 2)
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

func (m Model) renderSidebar() string {
	var rows []string
	for i, name := range TabNames {
		rows = append(rows, renderTabRow(name, "", Tab(i) == m.selectedTab))
	}
	return lipgloss.NewStyle().PaddingLeft(1).Render(
		lipgloss.JoinVertical(lipgloss.Left, rows...),
	)
}

func renderTabRow(name, subtitle string, selected bool) string {
	// Full inner layout: " name  [subtitle]  > "
	arrow := "> "
	right := arrow
	if subtitle != "" {
		right = subtitle + "  " + arrow
	}

	inner := SidebarWidth - 2 // 1 char padding on each side
	pad := inner - lipgloss.Width(name) - lipgloss.Width(right)
	if pad < 1 {
		pad = 1
	}
	spacer := strings.Repeat(" ", pad)

	if selected {
		full := " " + name + spacer + right + " "
		return tabSelectedStyle.Render(full)
	}

	namePart := tabNameStyle.Render(name)
	var subPart string
	if subtitle != "" {
		subPart = tabSubStyle.Render(subtitle) + "  "
	}
	arrowPart := tabArrowStyle.Render(arrow)
	return tabRowStyle.Render(" " + namePart + spacer + subPart + arrowPart + " ")
}

func (m Model) renderTabContent(width int) string {
	if width < 20 {
		width = 20
	}
	body := ""
	switch m.selectedTab {
	case TabThisDevice:
		body = m.renderThisDevice()
	case TabNetshares:
		body = m.renderNetshares()
	case TabSSH:
		body = m.renderSSH()
	case TabVPN:
		body = m.renderVPN()
	}
	return contentStyle.Width(width).Render(body)
}

func (m Model) renderThisDevice() string {
	var b strings.Builder

	b.WriteString(sectionTitleStyle.Render("Name"))
	b.WriteString("\n")
	b.WriteString(sectionValueStyle.Render(m.host.Hostname))
	b.WriteString("\n\n")

	b.WriteString(sectionTitleStyle.Render("IPs"))
	b.WriteString("\n")
	if len(m.host.IPs) == 0 {
		b.WriteString(headerMutedStyle.Render("(none)"))
	} else {
		for _, ip := range m.host.IPs {
			b.WriteString(sectionValueStyle.Render(ip))
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (m Model) renderNetshares() string {
	var b strings.Builder
	b.WriteString(sectionTitleStyle.Render("Shares"))
	b.WriteString("\n\n")

	if m.configErr != "" {
		b.WriteString(errorStyle.Render("Config error: "))
		b.WriteString(headerMutedStyle.Render(trimErr(m.configErr)))
		b.WriteString("\n\n")
		b.WriteString(headerMutedStyle.Render("Check ~/.config/yabinui/config.toml"))
		return b.String()
	}

	if len(m.netshares) == 0 {
		b.WriteString(headerMutedStyle.Render("No shares configured."))
		b.WriteString("\n\n")
		b.WriteString(headerMutedStyle.Render("Add entries to ~/.config/yabinui/config.toml"))
		return b.String()
	}

	for i, e := range m.netshares {
		b.WriteString(m.renderNetshareRow(i, e))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	hint := "press enter to toggle selected share"
	if m.focus == FocusSidebar {
		hint = "press → to focus"
	}
	b.WriteString(helpStyle.Render(hint))
	return b.String()
}

func (m Model) renderNetshareRow(idx int, e NetshareEntry) string {
	cursor := "  "
	if m.focus == FocusContent && m.selectedTab == TabNetshares && idx == m.selectedNetshare {
		cursor = tabArrowStyle.Render("> ")
	}

	var dot, label string
	switch e.State {
	case NSMounted:
		dot = lipgloss.NewStyle().Foreground(Success).Render("●")
		label = sectionValueStyle.Render("Connected")
	case NSUnmounted:
		dot = lipgloss.NewStyle().Foreground(Muted).Render("○")
		label = headerMutedStyle.Render("Not connected")
	case NSMounting:
		dot = lipgloss.NewStyle().Foreground(Primary).Render(m.spinner.View())
		label = connectingStyle.Render("Connecting...")
	case NSUnmounting:
		dot = lipgloss.NewStyle().Foreground(Primary).Render(m.spinner.View())
		label = connectingStyle.Render("Disconnecting...")
	case NSChecking:
		dot = lipgloss.NewStyle().Foreground(Muted).Render(m.spinner.View())
		label = headerMutedStyle.Render("Checking...")
	case NSError:
		dot = lipgloss.NewStyle().Foreground(Error).Render("✗")
		label = errorStyle.Render("Error")
	}

	name := sectionValueStyle.Render(e.Def.Name)
	// Align: cursor + name (padded) + dot + label
	namePadded := padRight(name, 18)
	row := cursor + namePadded + dot + "  " + label

	if e.State == NSError && e.ErrMsg != "" {
		row += "\n    " + headerMutedStyle.Render(trimErr(e.ErrMsg))
	}
	return row
}

func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s + " "
	}
	return s + strings.Repeat(" ", width-w)
}

func trimErr(s string) string {
	const max = 80
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}

func (m Model) renderVPN() string {
	var b strings.Builder
	b.WriteString(sectionTitleStyle.Render("Connections"))
	b.WriteString("\n\n")

	// Row 0: OpenVPN, state derived from m.state.
	b.WriteString(m.renderOpenVPNRow(0))
	b.WriteString("\n")

	// Rows 1..N: WireGuard tunnels.
	for i, e := range m.wgEntries {
		b.WriteString(m.renderWGRow(i+1, e))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	hint := "press enter to toggle selected vpn"
	if m.focus == FocusSidebar {
		hint = "press → to focus"
	}
	b.WriteString(helpStyle.Render(hint))
	return b.String()
}

func (m Model) renderOpenVPNRow(rowIdx int) string {
	cursor := "  "
	if m.focus == FocusContent && m.selectedTab == TabVPN && rowIdx == m.selectedVPN {
		cursor = tabArrowStyle.Render("> ")
	}

	var dot, label string
	switch m.state {
	case StateConnected:
		dot = lipgloss.NewStyle().Foreground(Success).Render("●")
		label = sectionValueStyle.Render("Connected")
	case StateDisconnected:
		dot = lipgloss.NewStyle().Foreground(Muted).Render("○")
		label = headerMutedStyle.Render("Not connected")
	case StateConnecting:
		dot = lipgloss.NewStyle().Foreground(Primary).Render(m.spinner.View())
		label = connectingStyle.Render("Connecting...")
	case StateDisconnecting:
		dot = lipgloss.NewStyle().Foreground(Primary).Render(m.spinner.View())
		label = connectingStyle.Render("Disconnecting...")
	case StateChecking:
		dot = lipgloss.NewStyle().Foreground(Muted).Render(m.spinner.View())
		label = headerMutedStyle.Render("Checking...")
	case StateError:
		dot = lipgloss.NewStyle().Foreground(Error).Render("✗")
		label = errorStyle.Render("Error")
	}

	name := sectionValueStyle.Render("drivelinevpn")
	kind := headerMutedStyle.Render("(openvpn)")
	namePadded := padRight(name, 18)
	return cursor + namePadded + dot + "  " + label + "  " + kind
}

func (m Model) renderWGRow(rowIdx int, e WGEntry) string {
	cursor := "  "
	if m.focus == FocusContent && m.selectedTab == TabVPN && rowIdx == m.selectedVPN {
		cursor = tabArrowStyle.Render("> ")
	}

	var dot, label string
	switch e.State {
	case WGUp:
		dot = lipgloss.NewStyle().Foreground(Success).Render("●")
		label = sectionValueStyle.Render("Connected")
	case WGDown:
		dot = lipgloss.NewStyle().Foreground(Muted).Render("○")
		label = headerMutedStyle.Render("Not connected")
	case WGBringingUp:
		dot = lipgloss.NewStyle().Foreground(Primary).Render(m.spinner.View())
		label = connectingStyle.Render("Connecting...")
	case WGBringingDown:
		dot = lipgloss.NewStyle().Foreground(Primary).Render(m.spinner.View())
		label = connectingStyle.Render("Disconnecting...")
	case WGChecking:
		dot = lipgloss.NewStyle().Foreground(Muted).Render(m.spinner.View())
		label = headerMutedStyle.Render("Checking...")
	case WGError:
		dot = lipgloss.NewStyle().Foreground(Error).Render("✗")
		label = errorStyle.Render("Error")
	}

	name := sectionValueStyle.Render(e.Name)
	kind := headerMutedStyle.Render("(wireguard)")
	namePadded := padRight(name, 18)
	row := cursor + namePadded + dot + "  " + label + "  " + kind
	if e.State == WGError && e.ErrMsg != "" {
		row += "\n    " + headerMutedStyle.Render(trimErr(e.ErrMsg))
	}
	return row
}

func (m Model) renderSSH() string {
	var b strings.Builder
	b.WriteString(sectionTitleStyle.Render("Sessions"))
	b.WriteString("\n\n")

	if m.configErr != "" {
		b.WriteString(errorStyle.Render("Config error: "))
		b.WriteString(headerMutedStyle.Render(trimErr(m.configErr)))
		b.WriteString("\n\n")
		b.WriteString(headerMutedStyle.Render("Check ~/.config/yabinui/config.toml"))
		return b.String()
	}

	if len(m.sshEntries) == 0 {
		b.WriteString(headerMutedStyle.Render("No SSH sessions configured."))
		b.WriteString("\n\n")
		b.WriteString(headerMutedStyle.Render("Add [[ssh]] entries to ~/.config/yabinui/config.toml"))
		return b.String()
	}

	for i, e := range m.sshEntries {
		b.WriteString(m.renderSSHRow(i, e))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	hint := "press enter to open selected session"
	if m.focus == FocusSidebar {
		hint = "press → to focus"
	}
	b.WriteString(helpStyle.Render(hint))
	return b.String()
}

func (m Model) renderSSHRow(idx int, e SSHEntry) string {
	cursor := "  "
	if m.focus == FocusContent && m.selectedTab == TabSSH && idx == m.selectedSSH {
		cursor = tabArrowStyle.Render("> ")
	}

	name := sectionValueStyle.Render(e.Def.Name)
	namePadded := padRight(name, 22)
	cmd := headerMutedStyle.Render(e.Def.Command)

	row := cursor + namePadded + cmd
	if e.ErrMsg != "" {
		row += "\n    " + errorStyle.Render("✗ ") + headerMutedStyle.Render(trimErr(e.ErrMsg))
	}
	return row
}

// renderFooter: right-aligned quit hint.
func (m Model) renderFooter(width int) string {
	hint := helpStyle.Render("press q to quit")
	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Right).
		PaddingRight(1).
		PaddingTop(1).
		Render(hint)
}
