package network

import "time"

const (
	DeviceStatusUnknown = "unknown"
	DeviceStatusTrusted = "trusted"
	DeviceStatusBlocked = "blocked"
	DeviceStatusIgnored = "ignored"
)

type Config struct {
	ARPPath       string
	DHCPLeasePath string
	DNSLogPath    string
	BlocklistPath string
}

type Device struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	IP        string    `json:"ip"`
	MAC       string    `json:"mac"`
	Interface string    `json:"interface"`
	Status    string    `json:"status"`
	Note      string    `json:"note"`
	FirstSeen time.Time `json:"firstSeen"`
	LastSeen  time.Time `json:"lastSeen"`
	Source    string    `json:"source"`
}

type Visit struct {
	Timestamp time.Time `json:"timestamp"`
	Domain    string    `json:"domain"`
	ClientIP  string    `json:"clientIp"`
	Device    string    `json:"device"`
	Action    string    `json:"action"`
	Source    string    `json:"source"`
}

type Overview struct {
	GeneratedAt        time.Time  `json:"generatedAt"`
	DevicesTotal       int        `json:"devicesTotal"`
	DevicesOnline      int        `json:"devicesOnline"`
	DevicesUnknown     int        `json:"devicesUnknown"`
	DevicesBlocked     int        `json:"devicesBlocked"`
	RecentVisits       int        `json:"recentVisits"`
	BlockedVisits      int        `json:"blockedVisits"`
	LiveSources        []Source   `json:"liveSources"`
	BlocklistPath      string     `json:"blocklistPath"`
	BlocklistUpdatedAt *time.Time `json:"blocklistUpdatedAt,omitempty"`
}

type Source struct {
	Name      string `json:"name"`
	Path      string `json:"path,omitempty"`
	Available bool   `json:"available"`
	Detail    string `json:"detail,omitempty"`
}

type Snapshot struct {
	Overview Overview `json:"overview"`
	Devices  []Device `json:"devices"`
	Visits   []Visit  `json:"visits"`
}

type UpdateDeviceRequest struct {
	Status string `json:"status"`
	Name   string `json:"name"`
	Note   string `json:"note"`
}
