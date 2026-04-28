package vpn

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
)

const ConfigPath = "/etc/openvpn/client/drivelinevpn.conf"

// CheckStatus returns true if OpenVPN is currently running
func CheckStatus() (bool, error) {
	cmd := exec.Command("pgrep", "-x", "openvpn")
	cmd.Stderr = io.Discard
	output, err := cmd.Output()
	if err != nil {
		// pgrep returns exit code 1 if no process found - this is not an error
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, err
	}
	return strings.TrimSpace(string(output)) != "", nil
}

// Connect starts OpenVPN as a daemon
func Connect() error {
	cmd := exec.Command("sudo", "openvpn", "--config", ConfigPath, "--daemon")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("openvpn: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// Disconnect stops the OpenVPN daemon
func Disconnect() error {
	cmd := exec.Command("sudo", "killall", "openvpn")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("killall openvpn: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
