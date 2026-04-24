# yabinui

A terminal UI for managing OpenVPN, WireGuard tunnels, CIFS netshares, and
saved SSH sessions from one place.

## Features

- OpenVPN connect/disconnect from the header (`.` toggle)
- WireGuard tunnel on/off list on the VPN tab
- CIFS netshare mount/umount with live status
- Saved SSH sessions that launch in a new terminal window
- Config-driven via `~/.config/yabinui/config.toml`

## Requirements

### OpenVPN

OpenVPN must be installed:

```bash
# Arch Linux
sudo pacman -S openvpn

# Debian/Ubuntu
sudo apt install openvpn
```

### VPN Configuration

This TUI expects your OpenVPN config at:
```
/etc/openvpn/client/drivelinevpn.conf
```

To change this path, edit `internal/vpn/openvpn.go`:
```go
const ConfigPath = "/etc/openvpn/client/your-config.conf"
```

### Credentials Setup

The TUI assumes your credentials are saved in the config (no interactive password prompt). To set this up:

1. Create a credentials file:
   ```bash
   sudo nano /etc/openvpn/client/auth.txt
   ```

2. Add your username and password (one per line):
   ```
   your_username
   your_password
   ```

3. Secure the file:
   ```bash
   sudo chmod 600 /etc/openvpn/client/auth.txt
   ```

4. Add this line to your `.conf` file:
   ```
   auth-user-pass /etc/openvpn/client/auth.txt
   ```

### Passwordless sudo (optional but recommended)

By default you'll be prompted for your sudo password every time you connect
or disconnect. To skip that, install a targeted sudoers rule that grants
passwordless access **only** to the two exact commands this TUI runs — not
blanket sudo.

A ready-to-install file is in the repo at `yabinui.sudoers`. It covers
openvpn (connect/disconnect) and CIFS mounts to any Driveline server under
your home directory (mount/umount):

```
yabin ALL=(root) NOPASSWD: /usr/bin/openvpn --config /etc/openvpn/client/drivelinevpn.conf --daemon
yabin ALL=(root) NOPASSWD: /usr/bin/killall openvpn
yabin ALL=(root) NOPASSWD: /usr/bin/mount -t cifs //*.drivelinebaseball.com/* /home/yabin/* -o credentials=/home/yabin/.smbcreds\,uid=1000\,gid=1000
yabin ALL=(root) NOPASSWD: /usr/bin/umount /home/yabin/*
yabin ALL=(root) NOPASSWD: /usr/bin/wg-quick up *
yabin ALL=(root) NOPASSWD: /usr/bin/wg-quick down *
```

The mount/umount rules use glob patterns so new netshares added to
`~/.config/yabinui/config.toml` don't require any sudoers changes, as long as
they mount a `*.drivelinebaseball.com` share to a path directly under your
home directory and use `~/.smbcreds` for credentials.

Install:

```bash
sudo cp yabinui.sudoers /etc/sudoers.d/yabinui
sudo chmod 0440 /etc/sudoers.d/yabinui
```

Notes:
- The command paths must match exactly what the TUI invokes. If `which openvpn`
  or `which killall` returns something other than `/usr/bin/...`, edit the file
  before installing.
- If you change `ConfigPath` in `internal/vpn/openvpn.go`, update the rule too —
  sudoers matches the full argument list.
- To remove: `sudo rm /etc/sudoers.d/yabinui`.

## Configuration

Netshares and SSH sessions live in `~/.config/yabinui/config.toml`. The TUI
writes a default file on first run; edit or extend it to add/remove entries.

```toml
# Optional. Terminal command prefix used to launch SSH sessions.
# Default: alacritty --hold -e
terminal = "alacritty --hold -e"

[[netshare]]
name = "fileshare"
remote = "//dc-theiastorage.drivelinebaseball.com/fileshare"
mount_point = "~/fileshare"
creds_file = "~/.smbcreds"

[[ssh]]
name = "dev-server"
command = "ssh devbox"      # respects your ~/.ssh/config aliases

[[wireguard]]
name = "dlexternal"         # resolves to /etc/wireguard/dlexternal.conf
```

Pressing `Enter` on an SSH entry spawns a new terminal window running the
command. Authentication relies on your SSH keys / ssh-agent — passwords are
not stored by yabinui.

Changes take effect on next launch.

## Building

Requires Go 1.21+.

```bash
go build -o yabinui .
```

Or with mise:
```bash
mise use go@latest
go build -o yabinui .
```

## Usage

```bash
./yabinui
```

### Controls

| Key | Action |
|-----|--------|
| `c` | Connect (when disconnected) |
| `d` | Disconnect (when connected) |
| `q` | Quit |
| `Enter` | Retry (on error) |

### Notes

- You'll be prompted for your sudo password when connecting/disconnecting — unless you installed the sudoers rule above, in which case it's silent
- The VPN continues running after you close the TUI
- Reopen the TUI to check status or disconnect
