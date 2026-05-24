package ports

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultPollInterval = 5 * time.Second
	listenStateHex      = "0A"
)

type Port struct {
	Host    string
	Network string
	Port    uint32
}

type Config struct {
	PollInterval  time.Duration
	ProcRoot      string
	ReservedPorts []uint16
}

type Monitor struct {
	pollInterval  time.Duration
	procFiles     []procNetFile
	reservedPorts map[uint16]struct{}
}

type procNetFile struct {
	network string
	path    string
}

func NewMonitor(cfg Config) *Monitor {
	procRoot := strings.TrimSpace(cfg.ProcRoot)
	if procRoot == "" {
		procRoot = "/proc"
	}

	pollInterval := cfg.PollInterval
	if pollInterval <= 0 {
		pollInterval = defaultPollInterval
	}

	reservedPorts := map[uint16]struct{}{
		9090: {},
	}
	for _, port := range cfg.ReservedPorts {
		reservedPorts[port] = struct{}{}
	}

	return &Monitor{
		pollInterval: pollInterval,
		procFiles: []procNetFile{
			{
				network: "tcp4",
				path:    filepath.Join(procRoot, "net", "tcp"),
			},
			{
				network: "tcp6",
				path:    filepath.Join(procRoot, "net", "tcp6"),
			},
		},
		reservedPorts: reservedPorts,
	}
}

func (m *Monitor) List() ([]Port, error) {
	var ports []Port
	for _, procFile := range m.procFiles {
		entries, err := parseProcNetFile(procFile.path, procFile.network, m.reservedPorts)
		if err != nil {
			return nil, err
		}
		ports = append(ports, entries...)
	}

	sort.Slice(ports, func(i, j int) bool {
		if ports[i].Port != ports[j].Port {
			return ports[i].Port < ports[j].Port
		}
		if ports[i].Network != ports[j].Network {
			return ports[i].Network < ports[j].Network
		}
		return ports[i].Host < ports[j].Host
	})
	return ports, nil
}

func (m *Monitor) PollInterval() time.Duration {
	return m.pollInterval
}

func parseProcNetFile(path, network string, reservedPorts map[uint16]struct{}) ([]Port, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	firstLine := true
	var ports []Port
	for scanner.Scan() {
		if firstLine {
			firstLine = false
			continue
		}

		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}
		if fields[3] != listenStateHex {
			continue
		}

		host, port, err := decodeLocalAddress(fields[1])
		if err != nil {
			return nil, fmt.Errorf("decode local address %q from %s: %w", fields[1], path, err)
		}
		if port < 1024 {
			continue
		}
		if _, blocked := reservedPorts[uint16(port)]; blocked {
			continue
		}

		ports = append(ports, Port{
			Host:    host,
			Network: network,
			Port:    uint32(port),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}
	return ports, nil
}

func decodeLocalAddress(raw string) (string, uint16, error) {
	addressHex, portHex, ok := strings.Cut(raw, ":")
	if !ok {
		return "", 0, fmt.Errorf("missing port separator")
	}

	portValue, err := strconv.ParseUint(portHex, 16, 16)
	if err != nil {
		return "", 0, fmt.Errorf("parse port: %w", err)
	}

	host, err := decodeHost(addressHex)
	if err != nil {
		return "", 0, err
	}
	return host, uint16(portValue), nil
}

func decodeHost(addressHex string) (string, error) {
	rawBytes, err := hex.DecodeString(addressHex)
	if err != nil {
		return "", fmt.Errorf("decode address hex: %w", err)
	}

	switch len(rawBytes) {
	case net.IPv4len:
		reversed := []byte{
			rawBytes[3],
			rawBytes[2],
			rawBytes[1],
			rawBytes[0],
		}
		return net.IP(reversed).String(), nil
	case net.IPv6len:
		decoded := make([]byte, net.IPv6len)
		for offset := 0; offset < len(rawBytes); offset += 4 {
			decoded[offset] = rawBytes[offset+3]
			decoded[offset+1] = rawBytes[offset+2]
			decoded[offset+2] = rawBytes[offset+1]
			decoded[offset+3] = rawBytes[offset]
		}
		return net.IP(decoded).String(), nil
	default:
		return "", fmt.Errorf("unexpected address length %d", len(rawBytes))
	}
}
