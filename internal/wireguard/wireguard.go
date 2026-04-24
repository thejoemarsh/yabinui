package wireguard

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// IsUp reports whether a wireguard interface of the given name exists.
// wg-quick creates the interface when bringing the tunnel up; its presence
// in /sys/class/net is our signal. No root needed.
func IsUp(name string) (bool, error) {
	_, err := os.Stat(filepath.Join("/sys/class/net", name))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// Up brings the tunnel up via wg-quick.
func Up(name string) error {
	cmd := exec.Command("sudo", "wg-quick", "up", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("wg-quick up: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Down takes the tunnel down via wg-quick.
func Down(name string) error {
	cmd := exec.Command("sudo", "wg-quick", "down", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("wg-quick down: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
