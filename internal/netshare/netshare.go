package netshare

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

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

// Mount mounts the remote CIFS share at MountPoint.
func (n Netshare) Mount() error {
	// Mounting an already-mounted point fails with CIFS "mount error(16):
	// Device or resource busy". Treat it as a no-op so a stale UI state can
	// never turn into a spurious error.
	if mounted, err := n.IsMounted(); err == nil && mounted {
		return nil
	}
	if err := os.MkdirAll(n.MountPoint, 0o755); err != nil {
		return fmt.Errorf("create mount point: %w", err)
	}
	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("lookup user: %w", err)
	}
	opts := fmt.Sprintf("credentials=%s,uid=%s,gid=%s", n.CredsFile, u.Uid, u.Gid)
	cmd := exec.Command("sudo", "mount", "-t", "cifs", n.RemotePath, n.MountPoint, "-o", opts)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mount: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Unmount unmounts MountPoint.
func (n Netshare) Unmount() error {
	cmd := exec.Command("sudo", "umount", n.MountPoint)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if isBusy(msg) {
			return fmt.Errorf("umount: %s is in use — close any shell, editor, or file manager sitting in it (fuser -vm %s)", n.MountPoint, n.MountPoint)
		}
		return fmt.Errorf("umount: %w: %s", err, msg)
	}
	return nil
}

// isBusy reports whether a mount/umount failure was EBUSY. CIFS surfaces this
// as "mount error(16)", util-linux as "target is busy".
func isBusy(msg string) bool {
	m := strings.ToLower(msg)
	return strings.Contains(m, "busy") || strings.Contains(m, "error(16)")
}
