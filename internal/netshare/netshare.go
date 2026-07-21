package netshare

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"syscall"
	"time"
)

// healthTimeout bounds the statfs probe used to detect a stale mount. Shares
// are mounted with "soft" so I/O errors out rather than blocking forever, but
// the timeout guards the case where it does not.
const healthTimeout = 5 * time.Second

// Netshare describes a single CIFS mount target.
type Netshare struct {
	Name       string
	RemotePath string // e.g. //host/share
	MountPoint string // absolute local path
	CredsFile  string
}

// IsMounted returns true if n.MountPoint appears in /proc/mounts.
func (n Netshare) IsMounted() (bool, error) {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return false, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 && fields[1] == n.MountPoint {
			return true, nil
		}
	}
	return false, scanner.Err()
}

// IsHealthy reports whether the mount actually responds. A share can remain in
// /proc/mounts long after its SMB session died — switching VPNs reroutes the
// server address, which kills the TCP session but leaves the mountpoint behind.
// IsMounted cannot see that; only touching the filesystem can.
func (n Netshare) IsHealthy() bool {
	done := make(chan error, 1) // buffered so the probe never leaks on timeout
	go func() {
		var st syscall.Statfs_t
		done <- syscall.Statfs(n.MountPoint, &st)
	}()
	select {
	case err := <-done:
		return err == nil
	case <-time.After(healthTimeout):
		return false
	}
}

// IsStale reports whether MountPoint is mounted but no longer reachable.
func (n Netshare) IsStale() bool {
	mounted, err := n.IsMounted()
	if err != nil || !mounted {
		return false
	}
	return !n.IsHealthy()
}

// Mount mounts the remote CIFS share at MountPoint.
func (n Netshare) Mount() error {
	// Mounting an already-mounted point fails with CIFS "mount error(16):
	// Device or resource busy", so handle that case here rather than letting
	// it surface as an error.
	if mounted, err := n.IsMounted(); err == nil && mounted {
		if n.IsHealthy() {
			return nil // already mounted and working
		}
		// Stale leftover (typically a VPN switch). Clear it first, otherwise
		// the mount below fails with error(16) against the dead mountpoint.
		if err := n.Unmount(); err != nil {
			return fmt.Errorf("clear stale mount: %w", err)
		}
	}
	if err := os.MkdirAll(n.MountPoint, 0o755); err != nil {
		return fmt.Errorf("create mount point: %w", err)
	}
	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("lookup user: %w", err)
	}
	opts := fmt.Sprintf("credentials=%s,uid=%s,gid=%s", n.CredsFile, u.Uid, u.Gid)
	out, err := sudo("mount", "-t", "cifs", n.RemotePath, n.MountPoint, "-o", opts)
	if err != nil {
		return fmt.Errorf("mount: %w: %s", err, out)
	}
	return nil
}

// Unmount unmounts MountPoint, escalating to a forced unmount when the share
// is stale rather than genuinely in use.
func (n Netshare) Unmount() error {
	msg, err := n.umount()
	if err == nil {
		return nil
	}
	if !isBusy(msg) {
		return fmt.Errorf("umount: %w: %s", err, msg)
	}

	// EBUSY has two very different causes, and they need opposite handling.
	// If the server still answers, a local process really is holding the
	// mount, and forcing would risk losing its buffered writes.
	if n.IsHealthy() {
		return fmt.Errorf("umount: %s is in use — close any shell, editor, or file manager sitting in it (fuser -vm %s)", n.MountPoint, n.MountPoint)
	}

	// Server unreachable: the SMB session is already gone (usually a VPN or
	// route change), so there is nothing left to flush. "-f" exists for
	// exactly this case.
	if msg, err := n.umount("-f"); err != nil {
		return fmt.Errorf("umount -f: %w: %s (stale mount; if it persists: sudo umount -l %s)", err, msg, n.MountPoint)
	}
	return nil
}

// umount shells out to umount with optional flags, returning trimmed output.
func (n Netshare) umount(flags ...string) (string, error) {
	args := append([]string{"umount"}, flags...)
	args = append(args, n.MountPoint)
	out, err := sudo(args...)
	return out, err
}

// sudo runs a command with -n so a missing sudoers rule fails immediately
// instead of blocking on a password prompt the TUI cannot display.
func sudo(args ...string) (string, error) {
	full := append([]string{"-n"}, args...)
	out, err := exec.Command("sudo", full...).CombinedOutput()
	msg := strings.TrimSpace(string(out))
	if err != nil && strings.Contains(msg, "password is required") {
		return msg, fmt.Errorf("not permitted by sudoers: sudo %s — install the updated yabinui.sudoers", strings.Join(args, " "))
	}
	return msg, err
}

// isBusy reports whether a mount/umount failure was EBUSY. CIFS surfaces this
// as "mount error(16)", util-linux as "target is busy".
func isBusy(msg string) bool {
	m := strings.ToLower(msg)
	return strings.Contains(m, "busy") || strings.Contains(m, "error(16)")
}
