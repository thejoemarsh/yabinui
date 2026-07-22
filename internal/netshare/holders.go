package netshare

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// closeGrace bounds how long we wait for a SIGTERM'd holder to exit before
// giving up on it. File managers close in well under a second; the budget is
// generous so a slow one does not get reported as a failure.
const closeGrace = 3 * time.Second

// fileManagers are processes we are willing to close automatically to free a
// mount. They browse or index the filesystem and hold no unsaved state, so
// terminating one loses nothing — unlike a shell, an editor, or a running copy.
// The desktop indexers are D-Bus activated and come back on their own.
var fileManagers = map[string]bool{
	// GUI
	"nautilus": true, "nemo": true, "thunar": true, "dolphin": true,
	"pcmanfm": true, "pcmanfm-qt": true, "caja": true, "spacefm": true,
	"io.elementary.files": true, "krusader": true, "doublecmd": true,
	// TUI
	"yazi": true, "ranger": true, "lf": true, "nnn": true, "mc": true,
	"vifm": true, "joshuto": true, "superfile": true, "spf": true, "broot": true,
	// Indexers that watch whatever the file manager opened
	"tracker-miner-fs-3": true, "tracker-extract-3": true, "localsearch-3": true,
	"baloo_file": true, "trackerd": true,
}

// comm is capped at 15 characters, so a longer process name never matches the
// table as written. Register the truncated form of each entry as well.
func init() {
	const commMax = 15
	for name := range fileManagers {
		if len(name) > commMax {
			fileManagers[name[:commMax]] = true
		}
	}
}

// Holder is a process keeping a mountpoint busy.
type Holder struct {
	PID  int
	Name string // /proc/<pid>/comm
	How  string // "cwd", "root", "open file", or "mapped file"
	Path string // the path under the mountpoint that it holds
}

// Closable reports whether Holder is a file browser we may terminate.
func (h Holder) Closable() bool { return fileManagers[h.Name] }

func (h Holder) String() string {
	return fmt.Sprintf("%s[%d] (%s: %s)", h.Name, h.PID, h.How, h.Path)
}

// Holders lists the processes holding mountPoint. Apart from one stat of the
// mountpoint itself it works entirely through /proc, so it never walks the
// share — "lsof +D" and friends do, and hang for minutes on a slow one. Call it
// only for a responsive mount; the stat can block on a dead server.
func Holders(mountPoint string) []Holder {
	mountPoint = strings.TrimSuffix(mountPoint, "/")
	dev := mountDev(mountPoint)
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	self := os.Getpid()

	var holders []Holder
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid == self {
			continue // not a process directory, or us
		}
		base := "/proc/" + e.Name()
		// Read comm first: it also confirms the process still exists.
		comm, err := os.ReadFile(base + "/comm")
		if err != nil {
			continue
		}
		name := strings.TrimSpace(string(comm))

		found := false
		for _, l := range []struct{ link, how string }{
			{base + "/cwd", "cwd"},
			{base + "/root", "root"},
		} {
			if p, ok := linkUnder(l.link, mountPoint); ok {
				holders = append(holders, Holder{PID: pid, Name: name, How: l.how, Path: p})
				found = true
				break
			}
		}
		if found {
			continue // one reason per process is enough to report
		}
		if p, how, ok := scanFDs(base, mountPoint, dev); ok {
			holders = append(holders, Holder{PID: pid, Name: name, How: how, Path: p})
			continue
		}
		if p, ok := mappedUnder(base+"/maps", mountPoint); ok {
			holders = append(holders, Holder{PID: pid, Name: name, How: "mapped file", Path: p})
		}
	}
	sort.Slice(holders, func(i, j int) bool { return holders[i].PID < holders[j].PID })
	return holders
}

// linkUnder resolves a /proc symlink and reports whether it points inside mp.
func linkUnder(link, mp string) (string, bool) {
	target, err := os.Readlink(link)
	if err != nil {
		return "", false // process gone, or not ours to inspect
	}
	return target, under(target, mp)
}

// scanFDs looks through a process's file descriptors for anything pinning mp:
// an open file inside it, or an inotify watch on it. The watch matters most —
// a GUI file manager showing a directory usually keeps no open descriptor and
// no cwd there, only a watch, and that alone is enough to make umount fail.
func scanFDs(procDir, mp string, dev uint64) (string, string, bool) {
	fdDir := procDir + "/fd"
	fds, err := os.ReadDir(fdDir)
	if err != nil {
		return "", "", false
	}
	for _, fd := range fds {
		target, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
		if err != nil {
			continue
		}
		if under(target, mp) {
			return target, "open file", true
		}
		if strings.Contains(target, "inotify") && watchesDev(procDir+"/fdinfo/"+fd.Name(), dev) {
			return mp, "watching it (file manager or sync tool)", true
		}
	}
	return "", "", false
}

// watchesDev reports whether an inotify fd holds a watch on the filesystem
// identified by dev. Each watch line in fdinfo carries the superblock device of
// the watched inode, and a mount has a superblock all to itself, so matching the
// device is equivalent to "this watch is inside that mount".
func watchesDev(fdinfo string, dev uint64) bool {
	if dev == 0 {
		return false
	}
	data, err := os.ReadFile(fdinfo)
	if err != nil {
		return false
	}
	want := "sdev:" + strconv.FormatUint(dev, 16)
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "inotify ") {
			continue
		}
		for _, f := range strings.Fields(line) {
			if f == want {
				return true
			}
		}
	}
	return false
}

// mountDev returns the device of a mountpoint in the encoding the kernel prints
// in fdinfo, which packs major/minor differently from the userspace dev_t that
// stat(2) hands back.
func mountDev(path string) uint64 {
	var st syscall.Stat_t
	if err := syscall.Stat(path, &st); err != nil {
		return 0
	}
	dev := uint64(st.Dev)
	major := ((dev >> 8) & 0xfff) | ((dev >> 32) &^ 0xfff)
	minor := (dev & 0xff) | ((dev >> 12) &^ 0xff)
	return major<<20 | minor
}

// mappedUnder scans a process's memory maps for a file inside mp. Media players
// and image viewers hold shares this way without an obvious open descriptor.
func mappedUnder(mapsFile, mp string) (string, bool) {
	data, err := os.ReadFile(mapsFile)
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(string(data), "\n") {
		// The path is the 6th field and may itself contain spaces, so cut at
		// the first "/" rather than splitting on whitespace.
		i := strings.Index(line, "/")
		if i < 0 {
			continue
		}
		if p := strings.TrimSuffix(line[i:], " (deleted)"); under(p, mp) {
			return p, true
		}
	}
	return "", false
}

// under reports whether path is mp itself or sits inside it. The separator
// check keeps "/home/yabin/fileshare-old" from matching "/home/yabin/fileshare".
func under(path, mp string) bool {
	return path == mp || strings.HasPrefix(path, mp+"/")
}

// closeHolders SIGTERMs every closable holder and waits for it to exit. It
// returns the holders it could not deal with: those we refuse to kill, plus any
// that ignored the signal.
func closeHolders(holders []Holder) []Holder {
	var remaining, signalled []Holder
	for _, h := range holders {
		if !h.Closable() {
			remaining = append(remaining, h)
			continue
		}
		if err := syscall.Kill(h.PID, syscall.SIGTERM); err != nil {
			remaining = append(remaining, h) // not ours, or already gone
			continue
		}
		signalled = append(signalled, h)
	}

	deadline := time.Now().Add(closeGrace)
	for len(signalled) > 0 && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
		var alive []Holder
		for _, h := range signalled {
			if running(h.PID) {
				alive = append(alive, h)
			}
		}
		signalled = alive
	}
	return append(remaining, signalled...)
}

// running reports whether a pid is still alive and holding resources. A zombie
// has released its mounts but lingers in the process table until its parent
// reaps it, and it still answers signal 0 — so check the state field instead.
func running(pid int) bool {
	stat, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return false
	}
	// Fields: pid (comm) state ... — comm is parenthesised and may contain
	// spaces, so start after the final ")".
	i := strings.LastIndex(string(stat), ")")
	if i < 0 || i+2 >= len(stat) {
		return false
	}
	return stat[i+2] != 'Z'
}

// describe renders holders for an error message, most useful detail first.
func describe(holders []Holder) string {
	parts := make([]string, 0, len(holders))
	for _, h := range holders {
		parts = append(parts, h.String())
	}
	return strings.Join(parts, ", ")
}
