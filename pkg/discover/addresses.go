// ABOUTME: Detects network addresses where the relay server is reachable.
// ABOUTME: Returns localhost, LAN, and Tailscale URLs for a given port.

package discover

import (
	"encoding/json"
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

	if host := tailscaleHost(); host != "" {
		add(host)
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

func tailscaleHost() string {
	out, err := exec.Command("tailscale", "status", "--json").Output()
	if err != nil {
		return ""
	}
	return parseTailscaleHost(out)
}

func parseTailscaleHost(data []byte) string {
	var status struct {
		Self struct {
			DNSName string `json:"DNSName"`
		} `json:"Self"`
	}
	if err := json.Unmarshal(data, &status); err != nil {
		return ""
	}
	return strings.TrimSuffix(strings.TrimSpace(status.Self.DNSName), ".")
}
