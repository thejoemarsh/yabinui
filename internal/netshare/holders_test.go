package netshare

import (
	"os/exec"
	"strings"
	"testing"
)

func TestUnder(t *testing.T) {
	mp := "/home/yabin/fileshare"
	cases := []struct {
		path string
		want bool
	}{
		{mp, true},
		{mp + "/Coaching/notes.md", true},
		{mp + "-old/notes.md", false}, // sibling with a shared prefix
		{"/home/yabin", false},
		{"/home/yabin/fileshareother", false},
	}
	for _, c := range cases {
		if got := under(c.path, mp); got != c.want {
			t.Errorf("under(%q, %q) = %v, want %v", c.path, mp, got, c.want)
		}
	}
}

// TestHoldersFindsCwd checks the /proc scan against a process we place in a
// known directory ourselves.
func TestHoldersFindsCwd(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("sleep", "60")
	cmd.Dir = dir
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() }()

	holders := Holders(dir)
	if len(holders) != 1 || holders[0].PID != cmd.Process.Pid || holders[0].How != "cwd" {
		t.Fatalf("Holders(%q) = %v, want the sleep process by cwd", dir, holders)
	}
}

// TestHoldersFindsWatcher covers the case that motivated the watch scan: a
// process pinning a directory with nothing but an inotify watch — no cwd, no
// open descriptor. That is how a GUI file manager holds a share it is showing.
func TestHoldersFindsWatcher(t *testing.T) {
	if _, err := exec.LookPath("inotifywait"); err != nil {
		t.Skip("inotifywait not installed")
	}
	dir := t.TempDir()
	cmd := exec.Command("inotifywait", "-m", dir)
	cmd.Dir = "/" // keep the watch as the only hold on dir
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() }()
	// inotifywait prints "Watches established." once the watch is in place.
	buf := make([]byte, 64)
	if _, err := stderr.Read(buf); err != nil {
		t.Fatalf("waiting for watch: %v", err)
	}

	holders := Holders(dir)
	if len(holders) != 1 || holders[0].PID != cmd.Process.Pid {
		t.Fatalf("Holders(%q) = %v, want the inotifywait process", dir, holders)
	}
	if !strings.Contains(holders[0].How, "watching") {
		t.Errorf("How = %q, want the watch reason", holders[0].How)
	}
}

// TestCloseHolders exercises the terminate-and-wait path. "sleep" stands in for
// a file manager so the test does not depend on one being installed.
func TestCloseHolders(t *testing.T) {
	fileManagers["sleep"] = true
	defer delete(fileManagers, "sleep")

	dir := t.TempDir()
	closable := exec.Command("sleep", "60")
	closable.Dir = dir
	if err := closable.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer func() { _ = closable.Process.Kill(); _, _ = closable.Process.Wait() }()

	// A shell in the same directory must survive: we never kill those.
	// The loop keeps bash from exec'ing into sleep and inheriting its name.
	shell := exec.Command("bash", "-c", "while :; do sleep 1; done")
	shell.Dir = dir
	if err := shell.Start(); err != nil {
		t.Fatalf("start shell: %v", err)
	}
	defer func() { _ = shell.Process.Kill(); _, _ = shell.Process.Wait() }()

	stuck := closeHolders(Holders(dir))
	if running(closable.Process.Pid) {
		t.Error("closable holder was not terminated")
	}
	if !running(shell.Process.Pid) {
		t.Error("bash was terminated; only file managers may be closed")
	}
	for _, h := range stuck {
		if h.Name == "sleep" {
			t.Errorf("closed holder still reported as stuck: %v", h)
		}
	}
	if len(stuck) == 0 {
		t.Error("expected the shell to be reported as a stuck holder")
	}
}
