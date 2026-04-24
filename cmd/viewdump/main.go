package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"yabinui/internal/tui"
)

func main() {
	m := tui.NewModel()
	var updated tea.Model = m
	updated, _ = updated.Update(tea.WindowSizeMsg{Width: 120, Height: 30})

	switch os.Getenv("DUMP_STATE") {
	case "connected":
		updated, _ = updated.Update(tui.StatusCheckedMsg{Connected: true})
	case "disconnected":
		updated, _ = updated.Update(tui.StatusCheckedMsg{Connected: false})
	}

	// Tab selection override for previewing different panes.
	if tab := os.Getenv("DUMP_TAB"); tab != "" {
		idx := map[string]int{"device": 0, "vpn": 1, "netshares": 2, "ssh": 3}[tab]
		for i := 0; i < idx; i++ {
			updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyDown})
		}
	}

	// Focus content pane if requested (right-arrow).
	if os.Getenv("DUMP_FOCUS") == "content" {
		updated, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRight})
	}

	// Seed netshare state for preview.
	switch os.Getenv("DUMP_NS") {
	case "mounted":
		updated, _ = updated.Update(tui.NetshareCheckedMsg{Idx: 0, Mounted: true})
	case "unmounted":
		updated, _ = updated.Update(tui.NetshareCheckedMsg{Idx: 0, Mounted: false})
	}

	fmt.Println(updated.View())
}
