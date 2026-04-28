package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// tabHasInteractiveContent reports whether the content pane for this tab
// accepts focus (i.e. has things to select / act on).
func tabHasInteractiveContent(t Tab) bool {
	switch t {
	case TabVPN, TabNetshares, TabSSH:
		return true
	}
	return false
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		first := m.width == 0
		m.width = msg.Width
		m.height = msg.Height
		if first {
			return m, tea.ClearScreen
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case StatusCheckedMsg:
		if msg.Err != nil {
			m.state = StateError
			m.errorMsg = msg.Err.Error()
			return m, nil
		}
		if msg.Connected {
			m.state = StateConnected
		} else {
			m.state = StateDisconnected
		}
		return m, nil

	case connectedMsg:
		if msg.err != nil {
			m.state = StateError
			m.errorMsg = msg.err.Error()
			return m, nil
		}
		m.state = StateConnected
		return m, refreshHostInfoCmd()

	case disconnectedMsg:
		if msg.err != nil {
			m.state = StateError
			m.errorMsg = msg.err.Error()
			return m, nil
		}
		m.state = StateDisconnected
		return m, refreshHostInfoCmd()

	case hostInfoRefreshedMsg:
		m.host = msg.host
		return m, nil

	case NetshareCheckedMsg:
		if msg.Idx < 0 || msg.Idx >= len(m.netshares) {
			return m, nil
		}
		if msg.Err != nil {
			m.netshares[msg.Idx].State = NSError
			m.netshares[msg.Idx].ErrMsg = msg.Err.Error()
			return m, nil
		}
		if msg.Mounted {
			m.netshares[msg.Idx].State = NSMounted
		} else {
			m.netshares[msg.Idx].State = NSUnmounted
		}
		m.netshares[msg.Idx].ErrMsg = ""
		return m, nil

	case netshareMountedMsg:
		if msg.idx < 0 || msg.idx >= len(m.netshares) {
			return m, nil
		}
		if msg.err != nil {
			m.netshares[msg.idx].State = NSError
			m.netshares[msg.idx].ErrMsg = msg.err.Error()
			return m, nil
		}
		m.netshares[msg.idx].State = NSMounted
		m.netshares[msg.idx].ErrMsg = ""
		return m, nil

	case sshLaunchedMsg:
		if msg.idx < 0 || msg.idx >= len(m.sshEntries) {
			return m, nil
		}
		if msg.err != nil {
			m.sshEntries[msg.idx].ErrMsg = msg.err.Error()
		}
		return m, nil

	case WGCheckedMsg:
		if msg.Idx < 0 || msg.Idx >= len(m.wgEntries) {
			return m, nil
		}
		if msg.Err != nil {
			m.wgEntries[msg.Idx].State = WGError
			m.wgEntries[msg.Idx].ErrMsg = msg.Err.Error()
			return m, nil
		}
		if msg.Up {
			m.wgEntries[msg.Idx].State = WGUp
		} else {
			m.wgEntries[msg.Idx].State = WGDown
		}
		m.wgEntries[msg.Idx].ErrMsg = ""
		return m, nil

	case wgUpMsg:
		if msg.idx < 0 || msg.idx >= len(m.wgEntries) {
			return m, nil
		}
		if msg.err != nil {
			m.wgEntries[msg.idx].State = WGError
			m.wgEntries[msg.idx].ErrMsg = msg.err.Error()
			return m, nil
		}
		m.wgEntries[msg.idx].State = WGUp
		m.wgEntries[msg.idx].ErrMsg = ""
		return m, refreshHostInfoCmd()

	case wgDownMsg:
		if msg.idx < 0 || msg.idx >= len(m.wgEntries) {
			return m, nil
		}
		if msg.err != nil {
			m.wgEntries[msg.idx].State = WGError
			m.wgEntries[msg.idx].ErrMsg = msg.err.Error()
			return m, nil
		}
		m.wgEntries[msg.idx].State = WGDown
		m.wgEntries[msg.idx].ErrMsg = ""
		return m, refreshHostInfoCmd()

	case netshareUnmountedMsg:
		if msg.idx < 0 || msg.idx >= len(m.netshares) {
			return m, nil
		}
		if msg.err != nil {
			m.netshares[msg.idx].State = NSError
			m.netshares[msg.idx].ErrMsg = msg.err.Error()
			return m, nil
		}
		m.netshares[msg.idx].State = NSUnmounted
		m.netshares[msg.idx].ErrMsg = ""
		return m, nil
	}

	return m, nil
}

func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {

	case "q", "ctrl+c":
		return m, tea.Quit

	case "up", "k":
		return m.moveUp(), nil

	case "down", "j":
		return m.moveDown(), nil

	case "right", "l":
		if m.focus == FocusSidebar && tabHasInteractiveContent(m.selectedTab) {
			m.focus = FocusContent
		}
		return m, nil

	case "enter":
		// In content focus on Netshares, enter toggles the selected share.
		if m.focus == FocusContent && m.selectedTab == TabNetshares {
			return m.toggleSelectedNetshare()
		}
		// In content focus on SSH, enter launches the selected session.
		if m.focus == FocusContent && m.selectedTab == TabSSH {
			return m.launchSelectedSSH()
		}
		// In content focus on VPN, enter toggles the selected vpn.
		if m.focus == FocusContent && m.selectedTab == TabVPN {
			return m.toggleSelectedVPN()
		}
		// From sidebar, enter moves into a tab with interactive content.
		if m.focus == FocusSidebar && tabHasInteractiveContent(m.selectedTab) {
			m.focus = FocusContent
			return m, nil
		}
		// Retry after an error.
		if m.state == StateError {
			m.state = StateChecking
			m.errorMsg = ""
			return m, tea.Batch(m.spinner.Tick, checkStatusCmd())
		}
		return m, nil

	case "left", "h", "esc":
		if m.focus == FocusContent {
			m.focus = FocusSidebar
		}
		return m, nil

	case ".":
		return m.toggleVPN()
	}

	return m, nil
}

// moveUp/moveDown are focus-aware: in sidebar they switch tabs, in content
// they move the item selection within the current tab.
func (m Model) moveUp() Model {
	if m.focus == FocusContent {
		switch m.selectedTab {
		case TabNetshares:
			if m.selectedNetshare > 0 {
				m.selectedNetshare--
			}
		case TabSSH:
			if m.selectedSSH > 0 {
				m.selectedSSH--
			}
		case TabVPN:
			if m.selectedVPN > 0 {
				m.selectedVPN--
			}
		}
		return m
	}
	if m.selectedTab > 0 {
		m.selectedTab--
	}
	return m
}

func (m Model) moveDown() Model {
	if m.focus == FocusContent {
		switch m.selectedTab {
		case TabNetshares:
			if m.selectedNetshare < len(m.netshares)-1 {
				m.selectedNetshare++
			}
		case TabSSH:
			if m.selectedSSH < len(m.sshEntries)-1 {
				m.selectedSSH++
			}
		case TabVPN:
			// 0 = openvpn, 1..len(wgEntries) = wg
			if m.selectedVPN < len(m.wgEntries) {
				m.selectedVPN++
			}
		}
		return m
	}
	if int(m.selectedTab) < len(TabNames)-1 {
		m.selectedTab++
	}
	return m
}

func (m Model) toggleVPN() (tea.Model, tea.Cmd) {
	switch m.state {
	case StateConnected:
		m.state = StateDisconnecting
		return m, tea.Batch(m.spinner.Tick, disconnectCmd())
	case StateDisconnected:
		m.state = StateConnecting
		return m, tea.Batch(m.spinner.Tick, connectCmd())
	}
	return m, nil
}

// toggleSelectedVPN dispatches the toggle based on which VPN row is selected:
// index 0 = openvpn, index 1..N maps to wgEntries[idx-1].
func (m Model) toggleSelectedVPN() (tea.Model, tea.Cmd) {
	if m.selectedVPN == 0 {
		return m.toggleVPN()
	}
	wgIdx := m.selectedVPN - 1
	if wgIdx < 0 || wgIdx >= len(m.wgEntries) {
		return m, nil
	}
	e := m.wgEntries[wgIdx]
	switch e.State {
	case WGUp:
		m.wgEntries[wgIdx].State = WGBringingDown
		return m, tea.Batch(m.spinner.Tick, wgDownCmd(wgIdx, e.Name))
	case WGDown, WGError:
		m.wgEntries[wgIdx].State = WGBringingUp
		m.wgEntries[wgIdx].ErrMsg = ""
		return m, tea.Batch(m.spinner.Tick, wgUpCmd(wgIdx, e.Name))
	}
	return m, nil
}

func (m Model) launchSelectedSSH() (tea.Model, tea.Cmd) {
	if m.selectedSSH < 0 || m.selectedSSH >= len(m.sshEntries) {
		return m, nil
	}
	m.sshEntries[m.selectedSSH].ErrMsg = ""
	return m, launchSSHCmd(m.selectedSSH, m.sshEntries[m.selectedSSH].Def, m.terminalCmd)
}

func (m Model) toggleSelectedNetshare() (tea.Model, tea.Cmd) {
	if m.selectedNetshare < 0 || m.selectedNetshare >= len(m.netshares) {
		return m, nil
	}
	e := m.netshares[m.selectedNetshare]
	switch e.State {
	case NSMounted:
		m.netshares[m.selectedNetshare].State = NSUnmounting
		return m, tea.Batch(m.spinner.Tick, unmountNetshareCmd(m.selectedNetshare, e.Def))
	case NSUnmounted, NSError:
		m.netshares[m.selectedNetshare].State = NSMounting
		m.netshares[m.selectedNetshare].ErrMsg = ""
		return m, tea.Batch(m.spinner.Tick, mountNetshareCmd(m.selectedNetshare, e.Def))
	}
	return m, nil
}
