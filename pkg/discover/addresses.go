// ABOUTME: Detects network addresses where the relay server is reachable.
// ABOUTME: Returns localhost, LAN, and Tailscale URLs for a given port.

package discover

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
)

// ListenAddresses returns all URLs where the relay is reachable on the given port.
// Always starts with localhost, followed by LAN IPs, then Tailscale IP if available.
func ListenAddresses(port int) []string {
	seen := make(map[string]bool)
	var addrs []string

	add := func(host string) {
		url := fmt.Sprintf("http://%s:%d", host, port)
		if !seen[url] {
			seen[url] = true
			addrs = append(addrs, url)
		}
	}

	add("localhost")

	for _, ip := range lanIPs() {
		add(ip)
	}

	if ip := tailscaleIP(); ip != "" {
		add(ip)
	}

	return addrs
}

func lanIPs() []string {
	var ips []string

	ifaces, err := net.Interfaces()
	if err != nil {
		return ips
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ip := ipNet.IP.To4()
			if ip == nil {
				continue
			}

			ips = append(ips, ip.String())
		}
	}

	return ips
}

func tailscaleIP() string {
	out, err := exec.Command("tailscale", "ip", "-4").Output()
	if err != nil {
		return ""
	}

	ip := strings.TrimSpace(string(out))
	if net.ParseIP(ip) == nil {
		return ""
	}

	return ip
}
