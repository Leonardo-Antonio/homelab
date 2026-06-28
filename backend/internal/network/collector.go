package network

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"
)

var (
	dnsmasqQueryPattern   = regexp.MustCompile(`query\[[A-Z0-9]+\]\s+([^\s]+)\s+from\s+([0-9a-fA-F:.]+)`)
	dnsmasqBlockedPattern = regexp.MustCompile(`(gravity blocked|blocked|reply error).*\s([a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`)
)

var dnsLogCandidates = []string{
	"/host/var/log/pihole/pihole.log",
	"/host/var/log/pihole.log",
	"/host/var/log/dnsmasq.log",
	"/var/log/pihole/pihole.log",
	"/var/log/pihole.log",
	"/var/log/dnsmasq.log",
}

type Collector struct {
	config Config
}

func NewCollector(config Config) *Collector {
	return &Collector{config: config}
}

func (c *Collector) Devices(ctx context.Context) ([]Device, []Source) {
	now := time.Now().UTC()
	devicesByID := map[string]Device{}
	sources := []Source{}

	arpDevices, arpSource := readARP(c.config.ARPPath, now)
	sources = append(sources, arpSource)
	for _, device := range arpDevices {
		devicesByID[device.ID] = device
	}

	leaseDevices, leaseSource := readDHCPLeases(c.config.DHCPLeasePath, now)
	if c.config.DHCPLeasePath != "" {
		sources = append(sources, leaseSource)
	}
	for _, device := range leaseDevices {
		existing, ok := devicesByID[device.ID]
		if ok {
			if existing.Name == existing.IP && device.Name != "" {
				existing.Name = device.Name
			}
			existing.Source = joinSource(existing.Source, device.Source)
			devicesByID[device.ID] = existing
			continue
		}
		devicesByID[device.ID] = device
	}

	devices := make([]Device, 0, len(devicesByID))
	for _, device := range devicesByID {
		devices = append(devices, device)
	}
	slices.SortFunc(devices, func(a, b Device) int {
		return strings.Compare(a.IP, b.IP)
	})
	return devices, sources
}

func (c *Collector) Visits(ctx context.Context, devices []Device, limit int) ([]Visit, Source) {
	visits, source := readDNSLog(c.config.DNSLogPath, devices, limit)
	return visits, source
}

func readARP(path string, now time.Time) ([]Device, Source) {
	if path == "" {
		return nil, Source{Name: "ARP", Available: false, Detail: "NETWORK_ARP_PATH vacío"}
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, Source{Name: "ARP", Path: path, Available: false, Detail: err.Error()}
	}
	defer file.Close()

	var devices []Device
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		if lineNumber == 1 {
			continue
		}
		fields := strings.Fields(scanner.Text())
		if len(fields) < 6 {
			continue
		}
		ip := fields[0]
		mac := strings.ToLower(fields[3])
		if mac == "00:00:00:00:00:00" || net.ParseIP(ip) == nil {
			continue
		}
		devices = append(devices, Device{
			ID:        idFromMAC(mac),
			Name:      ip,
			IP:        ip,
			MAC:       mac,
			Interface: fields[5],
			Status:    DeviceStatusUnknown,
			FirstSeen: now,
			LastSeen:  now,
			Source:    "ARP",
		})
	}
	if err := scanner.Err(); err != nil {
		return devices, Source{Name: "ARP", Path: path, Available: false, Detail: err.Error()}
	}
	return devices, Source{Name: "ARP", Path: path, Available: true, Detail: fmt.Sprintf("%d dispositivos", len(devices))}
}

func readDHCPLeases(path string, now time.Time) ([]Device, Source) {
	if path == "" {
		return nil, Source{Name: "DHCP", Available: false, Detail: "configura NETWORK_DHCP_LEASE_PATH"}
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, Source{Name: "DHCP", Path: path, Available: false, Detail: err.Error()}
	}
	defer file.Close()

	var devices []Device
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}
		mac := strings.ToLower(fields[1])
		ip := fields[2]
		name := fields[3]
		if name == "*" || name == "" {
			name = ip
		}
		if net.ParseIP(ip) == nil || !strings.Contains(mac, ":") {
			continue
		}
		devices = append(devices, Device{
			ID:        idFromMAC(mac),
			Name:      name,
			IP:        ip,
			MAC:       mac,
			Status:    DeviceStatusUnknown,
			FirstSeen: now,
			LastSeen:  now,
			Source:    "DHCP",
		})
	}
	if err := scanner.Err(); err != nil {
		return devices, Source{Name: "DHCP", Path: path, Available: false, Detail: err.Error()}
	}
	return devices, Source{Name: "DHCP", Path: path, Available: true, Detail: fmt.Sprintf("%d leases", len(devices))}
}

func readDNSLog(path string, devices []Device, limit int) ([]Visit, Source) {
	if path == "" {
		path = firstExistingPath(dnsLogCandidates)
	}
	if path == "" {
		return nil, Source{
			Name:      "DNS",
			Available: false,
			Detail:    "no encontré pihole.log ni dnsmasq.log; configura NETWORK_DNS_LOG_PATH",
		}
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, Source{Name: "DNS", Path: path, Available: false, Detail: err.Error()}
	}
	defer file.Close()

	nameByIP := map[string]string{}
	for _, device := range devices {
		nameByIP[device.IP] = device.Name
	}

	var visits []Visit
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		visit, ok := parseDNSLine(line, nameByIP)
		if !ok {
			continue
		}
		visits = append(visits, visit)
		if len(visits) > limit*3 {
			visits = visits[len(visits)-limit*2:]
		}
	}
	if err := scanner.Err(); err != nil {
		return visits, Source{Name: "DNS", Path: path, Available: false, Detail: err.Error()}
	}
	slices.Reverse(visits)
	if len(visits) > limit {
		visits = visits[:limit]
	}
	return visits, Source{Name: "DNS", Path: path, Available: true, Detail: fmt.Sprintf("%d eventos recientes", len(visits))}
}

func firstExistingPath(paths []string) string {
	for _, path := range paths {
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			return path
		}
	}
	return ""
}

func parseDNSLine(line string, nameByIP map[string]string) (Visit, bool) {
	match := dnsmasqQueryPattern.FindStringSubmatch(line)
	action := "allowed"
	if len(match) < 3 {
		blocked := dnsmasqBlockedPattern.FindStringSubmatch(line)
		if len(blocked) < 3 {
			return Visit{}, false
		}
		match = []string{"", blocked[2], ""}
		action = "blocked"
	}
	domain := strings.Trim(strings.ToLower(match[1]), ".")
	if domain == "" || !strings.Contains(domain, ".") {
		return Visit{}, false
	}
	clientIP := ""
	if len(match) > 2 {
		clientIP = match[2]
	}
	return Visit{
		Timestamp: time.Now().UTC(),
		Domain:    domain,
		ClientIP:  clientIP,
		Device:    nameByIP[clientIP],
		Action:    action,
		Source:    "DNS",
	}, true
}

func idFromMAC(mac string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(mac)), ":", "")
}

func joinSource(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" || strings.Contains(a, b) {
		return a
	}
	return a + "+" + b
}

func pathExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil || !errors.Is(err, os.ErrNotExist)
}
