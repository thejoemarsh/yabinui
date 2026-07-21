package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"yabinui/internal/config"
	"yabinui/internal/netshare"
	"yabinui/internal/ssh"
	"yabinui/internal/vpn"
	"yabinui/internal/wireguard"
)

const AppVersion = "v0.1.0"

// AppState represents the VPN connection state.
type AppState int

const (
	StateChecking AppState = iota
	StateDisconnected
	StateConnecting
	StateConnected
	StateDisconnecting
	StateError
)

// Tab identifies a sidebar entry.
type Tab int

const (
	TabThisDevice Tab = iota
	TabVPN
	TabNetshares
	TabSSH
)

var TabNames = []string{"This Device", "VPN", "Netshares", "SSH"}

// Focus tracks whether keyboard input drives the sidebar or the content pane.
type Focus int

const (
	FocusSidebar Focus = iota
	FocusContent
)

// NetshareState is the per-share mount state.
type NetshareState int

const (
	NSChecking NetshareState = iota
	NSUnmounted
	NSMounting
	NSMounted
	NSUnmounting
	NSError
	// NSStale means the mountpoint is still in /proc/mounts but the server
	// stopped answering — typically after switching VPNs.
	NSStale
)

// NetshareEntry pairs a share definition with its current state.
type NetshareEntry struct {
	Def    netshare.Netshare
	State  NetshareState
	ErrMsg string
}

// SSHEntry is a saved SSH session plus last-launch error state.
type SSHEntry struct {
	Def    ssh.Entry
	ErrMsg string
}

// WGState is the per-tunnel state for a WireGuard entry.
type WGState int

const (
	WGChecking WGState = iota
	WGDown
	WGBringingUp
	WGUp
	WGBringingDown
	WGError
)

// WGEntry pairs a wireguard tunnel name with its current state.
type WGEntry struct {
	Name   string
	State  WGState
	ErrMsg string
}

// Model is the main application state.
type Model struct {
	state    AppState
	spinner  spinner.Model
	errorMsg string

	selectedTab Tab
	focus       Focus
	host        HostInfo

	netshares        []NetshareEntry
	selectedNetshare int

	sshEntries   []SSHEntry
	selectedSSH  int
	terminalCmd  string

	wgEntries   []WGEntry
	selectedVPN int // 0 = openvpn, 1..N = wgEntries[selectedVPN-1]

	configErr string

	width  int
	height int
}

// --- Messages ---

type StatusCheckedMsg struct {
	Connected bool
	Err       error
}

type connectedMsg struct {
	err error
}

type disconnectedMsg struct {
	err error
}

type NetshareCheckedMsg struct {
	Idx     int
	Mounted bool
	Stale   bool // in /proc/mounts but the server no longer answers
	Err     error
}

type netshareMountedMsg struct {
	idx int
	err error
}

type netshareUnmountedMsg struct {
	idx int
	err error
}

type sshLaunchedMsg struct {
	idx int
	err error
}

type WGCheckedMsg struct {
	Idx int
	Up  bool
	Err error
}

type wgUpMsg struct {
	idx int
	err error
}

type wgDownMsg struct {
	idx int
	err error
}

type hostInfoRefreshedMsg struct {
	host HostInfo
}

// DrivelineSpinner is a custom spinner with smooth animation
var DrivelineSpinner = spinner.Spinner{
	Frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	FPS:    time.Second / 12,
}

// NewModel creates and initializes the application model
func NewModel() Model {
	s := spinner.New()
	s.Spinner = DrivelineSpinner
	s.Style = lipgloss.NewStyle().Foreground(Primary)

	var shares []NetshareEntry
	var sshList []SSHEntry
	var cfgErr string
	terminal := config.DefaultTerminal

	cfg, err := config.Load()
	if err != nil {
		cfgErr = err.Error()
	} else {
		terminal = cfg.Terminal
		for _, nc := range cfg.Netshares {
			shares = append(shares, NetshareEntry{
				Def: netshare.Netshare{
					Name:       nc.Name,
					RemotePath: nc.Remote,
					MountPoint: nc.MountPoint,
					CredsFile:  nc.CredsFile,
				},
				State: NSChecking,
			})
		}
		for _, sc := range cfg.SSH {
			sshList = append(sshList, SSHEntry{
				Def: ssh.Entry{Name: sc.Name, Command: sc.Command},
			})
		}
	}

	var wgList []WGEntry
	if cfg != nil {
		for _, wc := range cfg.Wireguards {
			wgList = append(wgList, WGEntry{Name: wc.Name, State: WGChecking})
		}
	}

	return Model{
		state:       StateChecking,
		spinner:     s,
		selectedTab: TabThisDevice,
		focus:       FocusSidebar,
		host:        LoadHostInfo(),
		netshares:   shares,
		sshEntries:  sshList,
		terminalCmd: terminal,
		wgEntries:   wgList,
		configErr:   cfgErr,
	}
}

// Init returns the initial command to run
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.spinner.Tick,
		checkStatusCmd(),
	}
	for i, e := range m.netshares {
		cmds = append(cmds, checkNetshareCmd(i, e.Def))
	}
	for i, e := range m.wgEntries {
		cmds = append(cmds, checkWGCmd(i, e.Name))
	}
	return tea.Batch(cmds...)
}

// --- Commands ---

func checkStatusCmd() tea.Cmd {
	return func() tea.Msg {
		connected, err := vpn.CheckStatus()
		return StatusCheckedMsg{Connected: connected, Err: err}
	}
}

func connectCmd() tea.Cmd {
	return func() tea.Msg {
		err := vpn.Connect()
		return connectedMsg{err: err}
	}
}

func disconnectCmd() tea.Cmd {
	return func() tea.Msg {
		err := vpn.Disconnect()
		return disconnectedMsg{err: err}
	}
}

func checkNetshareCmd(idx int, n netshare.Netshare) tea.Cmd {
	return func() tea.Msg {
		mounted, err := n.IsMounted()
		stale := false
		if err == nil && mounted {
			stale = !n.IsHealthy()
		}
		return NetshareCheckedMsg{Idx: idx, Mounted: mounted, Stale: stale, Err: err}
	}
}

func mountNetshareCmd(idx int, n netshare.Netshare) tea.Cmd {
	return func() tea.Msg {
		err := n.Mount()
		return netshareMountedMsg{idx: idx, err: err}
	}
}

func unmountNetshareCmd(idx int, n netshare.Netshare) tea.Cmd {
	return func() tea.Msg {
		err := n.Unmount()
		return netshareUnmountedMsg{idx: idx, err: err}
	}
}

// remountNetshareCmd clears a stale mount and mounts it again. Mount() drops
// the dead mountpoint itself, so this is a plain reconnect.
func remountNetshareCmd(idx int, n netshare.Netshare) tea.Cmd {
	return func() tea.Msg {
		return netshareMountedMsg{idx: idx, err: n.Mount()}
	}
}

// resolveNetshareCmd re-reads /proc/mounts and toggles based on what is
// actually mounted, rather than on a cached (possibly stale) UI state. Used to
// recover from NSError, where the last action failed and the tracked state and
// reality may have diverged.
func resolveNetshareCmd(idx int, n netshare.Netshare) tea.Cmd {
	return func() tea.Msg {
		mounted, err := n.IsMounted()
		if err != nil {
			return netshareUnmountedMsg{idx: idx, err: err}
		}
		if mounted {
			return netshareUnmountedMsg{idx: idx, err: n.Unmount()}
		}
		return netshareMountedMsg{idx: idx, err: n.Mount()}
	}
}

func launchSSHCmd(idx int, e ssh.Entry, terminal string) tea.Cmd {
	return func() tea.Msg {
		err := e.Launch(terminal)
		return sshLaunchedMsg{idx: idx, err: err}
	}
}

func checkWGCmd(idx int, name string) tea.Cmd {
	return func() tea.Msg {
		up, err := wireguard.IsUp(name)
		return WGCheckedMsg{Idx: idx, Up: up, Err: err}
	}
}

func wgUpCmd(idx int, name string) tea.Cmd {
	return func() tea.Msg {
		err := wireguard.Up(name)
		return wgUpMsg{idx: idx, err: err}
	}
}

func wgDownCmd(idx int, name string) tea.Cmd {
	return func() tea.Msg {
		err := wireguard.Down(name)
		return wgDownMsg{idx: idx, err: err}
	}
}

func refreshHostInfoCmd() tea.Cmd {
	return func() tea.Msg {
		return hostInfoRefreshedMsg{host: LoadHostInfo()}
	}
}
