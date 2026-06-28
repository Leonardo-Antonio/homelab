package network

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) UpsertDiscovered(ctx context.Context, devices []Device) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, device := range devices {
		if device.ID == "" {
			continue
		}
		firstSeen := device.FirstSeen.UTC().Format(time.RFC3339Nano)
		lastSeen := device.LastSeen.UTC().Format(time.RFC3339Nano)
		_, err := r.db.ExecContext(ctx, `
			INSERT INTO network_devices (id, name, ip, mac, status, note, first_seen, last_seen, updated_at)
			VALUES (?, ?, ?, ?, ?, '', ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				name = CASE
					WHEN network_devices.name = '' OR network_devices.name = network_devices.ip THEN excluded.name
					ELSE network_devices.name
				END,
				ip = excluded.ip,
				mac = excluded.mac,
				last_seen = excluded.last_seen,
				updated_at = excluded.updated_at`,
			device.ID, device.Name, device.IP, device.MAC, DeviceStatusUnknown, firstSeen, lastSeen, now)
		if err != nil {
			return fmt.Errorf("upsert network device: %w", err)
		}
	}
	return nil
}

func (r *Repository) ListDecisions(ctx context.Context) (map[string]Device, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, ip, mac, status, note, first_seen, last_seen
		FROM network_devices`)
	if err != nil {
		return nil, fmt.Errorf("query network devices: %w", err)
	}
	defer rows.Close()

	devices := map[string]Device{}
	for rows.Next() {
		device, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}
		devices[device.ID] = device
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate network devices: %w", err)
	}
	return devices, nil
}

func (r *Repository) Update(ctx context.Context, id string, patch UpdateDeviceRequest) (Device, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := r.db.ExecContext(ctx, `
		UPDATE network_devices
		SET status = COALESCE(NULLIF(?, ''), status),
			name = COALESCE(NULLIF(?, ''), name),
			note = ?,
			updated_at = ?
		WHERE id = ?`,
		patch.Status, strings.TrimSpace(patch.Name), strings.TrimSpace(patch.Note), now, id)
	if err != nil {
		return Device{}, fmt.Errorf("update network device: %w", err)
	}
	return r.Get(ctx, id)
}

func (r *Repository) Get(ctx context.Context, id string) (Device, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, ip, mac, status, note, first_seen, last_seen
		FROM network_devices
		WHERE id = ?`, id)
	return scanDevice(row)
}

type deviceScanner interface {
	Scan(dest ...any) error
}

func scanDevice(scanner deviceScanner) (Device, error) {
	var device Device
	var firstSeen string
	var lastSeen string
	if err := scanner.Scan(&device.ID, &device.Name, &device.IP, &device.MAC, &device.Status, &device.Note, &firstSeen, &lastSeen); err != nil {
		return Device{}, err
	}
	parsedFirstSeen, err := time.Parse(time.RFC3339Nano, firstSeen)
	if err != nil {
		return Device{}, fmt.Errorf("parse network device first_seen: %w", err)
	}
	parsedLastSeen, err := time.Parse(time.RFC3339Nano, lastSeen)
	if err != nil {
		return Device{}, fmt.Errorf("parse network device last_seen: %w", err)
	}
	device.FirstSeen = parsedFirstSeen.UTC()
	device.LastSeen = parsedLastSeen.UTC()
	return device, nil
}
