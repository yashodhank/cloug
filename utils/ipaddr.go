package utils

import "fmt"
import "net"
import "strings"

var privateNetworks []*net.IPNet

func Init() {
	privateNetworks = make([]*net.IPNet, 0)
	for _, cidr := range []string{"192.168.0.0/16", "172.16.0.0/12", "10.0.0.0/8"} {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(err)
		}
		privateNetworks = append(privateNetworks, network)
	}
}

func IsPrivate(ipString string) bool {
	if privateNetworks == nil {
		Init()
	}

	ip := net.ParseIP(ipString)
	for _, net := range privateNetworks {
		if net.Contains(ip) {
			return true
		}
	}
	return false
}

// Returns the IP version of the specified IP address (e.g. 4 for IPv4, 6 for IPv6),
//  or 0 if the address is invalid.
func GetIPVersion(ipString string) int {
	ip := net.ParseIP(ipString)
	if ip == nil {
		return 0
	}
	if ip.To4() != nil {
		return 4
	} else {
		return 6
	}
}

func ParseCIDROrIP(s string) *net.IPNet {
	// first try as network
	_, network, err := net.ParseCIDR(s)
	if err == nil {
		return network
	}

	// else try as IP
	_, network, err = net.ParseCIDR(s + "/32")
	if err == nil {
		return network
	} else {
		return nil
	}
}

func ParseNetworks(s string) ([]*net.IPNet, error) {
	var networks []*net.IPNet
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			network := ParseCIDROrIP(s)
			if network != nil {
				networks = append(networks, network)
			} else {
				return nil, fmt.Errorf("failed to parse \"%s\" as IP/CIDR", part)
			}
		}
	}
	return networks, nil
}

func MatchNetworks(netString string, ipString string) bool {
	ip := net.ParseIP(ipString)
	if ip == nil {
		return false
	}

	networks, err := ParseNetworks(netString)
	if err != nil {
		return false
	}

	for _, network := range networks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
