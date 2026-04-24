package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// DefaultTerminal is the command prefix used to launch SSH sessions when
// the user hasn't overridden it in config. The SSH command is appended.
const DefaultTerminal = "alacritty --hold -e"

// Config is the on-disk user configuration.
type Config struct {
	Terminal   string          `toml:"terminal"`
	Netshares  []NetshareCfg   `toml:"netshare"`
	SSH        []SSHCfg        `toml:"ssh"`
	Wireguards []WireguardCfg  `toml:"wireguard"`
}

type NetshareCfg struct {
	Name       string `toml:"name"`
	Remote     string `toml:"remote"`
	MountPoint string `toml:"mount_point"`
	CredsFile  string `toml:"creds_file"`
}

type SSHCfg struct {
	Name    string `toml:"name"`
	Command string `toml:"command"`
}

type WireguardCfg struct {
	Name string `toml:"name"`
}

// Path returns the path to the user's config file (~/.config/yabinui/config.toml).
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "yabinui", "config.toml"), nil
}

// Load reads the config, writing a default file on first run.
func Load() (*Config, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(p); os.IsNotExist(err) {
		if err := writeDefault(p); err != nil {
			return nil, fmt.Errorf("write default config: %w", err)
		}
	}
	var cfg Config
	if _, err := toml.DecodeFile(p, &cfg); err != nil {
		return nil, fmt.Errorf("decode %s: %w", p, err)
	}
	cfg.expand()
	return &cfg, nil
}

func writeDefault(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(defaultTOML), 0o644)
}

// expand resolves ~/ prefixes and applies defaults.
func (c *Config) expand() {
	if strings.TrimSpace(c.Terminal) == "" {
		c.Terminal = DefaultTerminal
	}
	for i := range c.Netshares {
		c.Netshares[i].MountPoint = expandHome(c.Netshares[i].MountPoint)
		c.Netshares[i].CredsFile = expandHome(c.Netshares[i].CredsFile)
	}
}

func expandHome(p string) string {
	if p == "" {
		return p
	}
	if p == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	if strings.HasPrefix(p, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, p[2:])
	}
	return p
}

const defaultTOML = `# yabinui config
#
# Entries added here are picked up next time you launch the TUI.
# Paths may start with "~/" and will be expanded to your home directory.

# Terminal command used when launching SSH sessions. The SSH command
# gets appended as the final argument(s). Default: alacritty --hold -e
# terminal = "alacritty --hold -e"

[[netshare]]
name = "fileshare"
remote = "//dc-theiastorage.drivelinebaseball.com/fileshare"
mount_point = "~/fileshare"
creds_file = "~/.smbcreds"

[[netshare]]
name = "fileshare-dl"
remote = "//dc-storageserver.drivelinebaseball.com/fileshare"
mount_point = "~/fileshare-dl"
creds_file = "~/.smbcreds"

# SSH sessions. Each entry opens in a new terminal window on Enter.
# The "command" field is a raw shell-style ssh invocation — respect your
# ~/.ssh/config aliases, add flags, anything that works on the command line.
#
# [[ssh]]
# name = "dev-server"
# command = "ssh devbox"

[[ssh]]
name = "yabin-desk"
command = "ssh yabin@172.16.51.61"

[[ssh]]
name = "wa-blackburst"
command = "ssh produser@wa-blackburst.drivelinebaseball.com"

# WireGuard tunnels. "name" is the /etc/wireguard/<name>.conf basename.
# Toggled via sudo wg-quick up/down.
#
# [[wireguard]]
# name = "dlexternal"
`
