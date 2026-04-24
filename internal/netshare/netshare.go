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
		return fmt.Errorf("umount: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
