package systemd

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/coreos/go-systemd/v22/dbus"
)

type Mode int

const (
	SystemMode Mode = iota
	UserMode
)

func (m Mode) String() string {
	if m == SystemMode {
		return "System"
	}
	return "User"
}

type ServiceInfo struct {
	Name                       string
	Description                string
	LoadState                  string
	ActiveState                string
	SubState                   string
	UnitFileState              string // enabled, disabled, static, masked, etc.
	MainPID                    uint32
	MemoryCurrent              uint64
	MemoryLimit                uint64
	CPUUsageNSec               uint64
	TasksCurrent               uint64
	TasksMax                   uint64
	ActiveEnterTimestamp       uint64
	ActiveExitTimestamp        uint64
	ExecMainCode               int32
	ExecMainStatus             int32
	IPTrafficRxBytes           uint64
	IPTrafficTxBytes           uint64
	IOReadBytes                uint64
	IOWriteBytes               uint64
}

type Client struct {
	conn *dbus.Conn
	mode Mode
}

func NewClient(requestedMode *Mode) (*Client, error) {
	var conn *dbus.Conn
	var err error
	var finalMode Mode

	if requestedMode != nil {
		finalMode = *requestedMode
		if finalMode == SystemMode {
			conn, err = dbus.NewSystemdConnection()
		} else {
			conn, err = dbus.NewUserConnection()
		}
		if err != nil {
			return nil, err
		}
	} else {
		// Auto-detect
		conn, err = dbus.NewSystemdConnection()
		if err != nil {
			// Fallback to user bus
			conn, err = dbus.NewUserConnection()
			if err != nil {
				return nil, fmt.Errorf("failed to connect to system or user systemd dbus: %w", err)
			}
			finalMode = UserMode
		} else {
			finalMode = SystemMode
		}
	}

	return &Client{
		conn: conn,
		mode: finalMode,
	}, nil
}

func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *Client) Mode() Mode {
	return c.mode
}

// ListServices fetches all service units and merges their active states and enablement states.
func (c *Client) ListServices(ctx context.Context) ([]ServiceInfo, error) {
	// 1. Fetch active units from ListUnits
	units, err := c.conn.ListUnitsContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list units: %w", err)
	}

	// Map to accumulate services by name
	servicesMap := make(map[string]*ServiceInfo)

	for _, u := range units {
		if !strings.HasSuffix(u.Name, ".service") {
			continue
		}

		servicesMap[u.Name] = &ServiceInfo{
			Name:        u.Name,
			Description: u.Description,
			LoadState:   u.LoadState,
			ActiveState: u.ActiveState,
			SubState:    u.SubState,
		}
	}

	// 2. Fetch all unit files to get their enablement status
	unitFiles, err := c.conn.ListUnitFilesContext(ctx)
	if err == nil {
		for _, uf := range unitFiles {
			name := filepath.Base(uf.Path)
			if !strings.HasSuffix(name, ".service") {
				continue
			}

			if info, exists := servicesMap[name]; exists {
				info.UnitFileState = uf.Type
			} else {
				// Unit is not currently loaded/active but exists in system
				servicesMap[name] = &ServiceInfo{
					Name:          name,
					Description:   "",
					LoadState:     "not-loaded",
					ActiveState:   "inactive",
					SubState:      "dead",
					UnitFileState: uf.Type,
				}
			}
		}
	}

	// Convert map to slice
	services := make([]ServiceInfo, 0, len(servicesMap))
	for _, s := range servicesMap {
		services = append(services, *s)
	}

	return services, nil
}

// GetServiceDetails fetches detailed properties for a specific service.
func (c *Client) GetServiceDetails(ctx context.Context, name string) (*ServiceInfo, error) {
	props, err := c.conn.GetUnitPropertiesContext(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get unit properties: %w", err)
	}

	info := &ServiceInfo{
		Name: name,
	}

	if val, ok := props["Description"].(string); ok {
		info.Description = val
	}
	if val, ok := props["LoadState"].(string); ok {
		info.LoadState = val
	}
	if val, ok := props["ActiveState"].(string); ok {
		info.ActiveState = val
	}
	if val, ok := props["SubState"].(string); ok {
		info.SubState = val
	}
	if val, ok := props["UnitFileState"].(string); ok {
		info.UnitFileState = val
	}

	if val, ok := props["ActiveEnterTimestamp"].(uint64); ok {
		info.ActiveEnterTimestamp = val
	}
	if val, ok := props["ActiveExitTimestamp"].(uint64); ok {
		info.ActiveExitTimestamp = val
	}

	// Fetch service-specific properties
	sProps, err := c.conn.GetUnitTypePropertiesContext(ctx, name, "Service")
	if err == nil {
		if val, ok := sProps["MainPID"].(uint32); ok {
			info.MainPID = val
		}
		if val, ok := sProps["MemoryCurrent"].(uint64); ok {
			info.MemoryCurrent = val
		}
		if val, ok := sProps["MemoryLimit"].(uint64); ok {
			info.MemoryLimit = val
		}
		if val, ok := sProps["CPUUsageNSec"].(uint64); ok {
			info.CPUUsageNSec = val
		}
		if val, ok := sProps["TasksCurrent"].(uint64); ok {
			info.TasksCurrent = val
		}
		if val, ok := sProps["TasksMax"].(uint64); ok {
			info.TasksMax = val
		}
		if val, ok := sProps["ExecMainCode"].(int32); ok {
			info.ExecMainCode = val
		}
		if val, ok := sProps["ExecMainStatus"].(int32); ok {
			info.ExecMainStatus = val
		}
		if val, ok := sProps["IPTrafficRxBytes"].(uint64); ok {
			info.IPTrafficRxBytes = val
		}
		if val, ok := sProps["IPTrafficTxBytes"].(uint64); ok {
			info.IPTrafficTxBytes = val
		}
		if val, ok := sProps["IOReadBytes"].(uint64); ok {
			info.IOReadBytes = val
		}
		if val, ok := sProps["IOWriteBytes"].(uint64); ok {
			info.IOWriteBytes = val
		}
	}

	return info, nil
}

// Actions
func (c *Client) StartService(ctx context.Context, name string) error {
	ch := make(chan string, 1)
	_, err := c.conn.StartUnitContext(ctx, name, "replace", ch)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) StopService(ctx context.Context, name string) error {
	ch := make(chan string, 1)
	_, err := c.conn.StopUnitContext(ctx, name, "replace", ch)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) RestartService(ctx context.Context, name string) error {
	ch := make(chan string, 1)
	_, err := c.conn.RestartUnitContext(ctx, name, "replace", ch)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) EnableService(ctx context.Context, name string) error {
	_, _, err := c.conn.EnableUnitFilesContext(ctx, []string{name}, false, false)
	return err
}

func (c *Client) DisableService(ctx context.Context, name string) error {
	_, err := c.conn.DisableUnitFilesContext(ctx, []string{name}, false)
	return err
}

// GetLogs fetches the recent logs of a service using journalctl.
func (c *Client) GetLogs(ctx context.Context, name string, limit int) (string, error) {
	args := []string{}
	if c.mode == UserMode {
		args = append(args, "--user")
	}
	args = append(args, "-u", name, "-n", strconv.Itoa(limit), "--no-pager")

	cmd := exec.CommandContext(ctx, "journalctl", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("journalctl failed: %w (output: %s)", err, string(output))
	}
	return string(output), nil
}
