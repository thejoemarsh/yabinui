package tui

import (
	"net"
	"os"
	"os/user"
)

type HostInfo struct {
	Hostname string
	Username string
	IPs      []string
}

func LoadHostInfo() HostInfo {
	info := HostInfo{}

	if h, err := os.Hostname(); err == nil {
		info.Hostname = h
	} else {
		info.Hostname = "unknown"
	}

	if u, err := user.Current(); err == nil {
		info.Username = u.Username
	} else {
		info.Username = os.Getenv("USER")
	}

	info.IPs = collectIPs()
	return info
}

func collectIPs() []string {
	var ips []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return ips
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipnet.IP
			if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
				continue
			}
			ips = append(ips, ip.String())
		}
	}
	return ips
}
