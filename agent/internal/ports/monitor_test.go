package ports

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMonitorListParsesAndFiltersListeningPorts(t *testing.T) {
	t.Parallel()

	procRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(procRoot, "net"), 0o755); err != nil {
		t.Fatalf("create proc net dir: %v", err)
	}

	tcpContents := `  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:1F90 00000000:0000 0A 00000000:00000000 00:00000000 00000000   100        0 0 1 0000000000000000 100 0 0 10 0
   1: 0100007F:01BB 00000000:0000 0A 00000000:00000000 00:00000000 00000000   100        0 0 1 0000000000000000 100 0 0 10 0
   2: 00000000:2382 00000000:0000 0A 00000000:00000000 00:00000000 00000000   100        0 0 1 0000000000000000 100 0 0 10 0
   3: 0100007F:2000 00000000:0000 01 00000000:00000000 00:00000000 00000000   100        0 0 10 0
`
	tcp6Contents := `  sl  local_address                         rem_address                          st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 00000000000000000000000001000000:0BB8 00000000000000000000000000000000:0000 0A 00000000:00000000 00:00000000 00000000   100        0 0 1 0000000000000000 100 0 0 10 0
`

	if err := os.WriteFile(filepath.Join(procRoot, "net", "tcp"), []byte(tcpContents), 0o644); err != nil {
		t.Fatalf("write tcp fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(procRoot, "net", "tcp6"), []byte(tcp6Contents), 0o644); err != nil {
		t.Fatalf("write tcp6 fixture: %v", err)
	}

	monitor := NewMonitor(Config{ProcRoot: procRoot})
	ports, err := monitor.List()
	if err != nil {
		t.Fatalf("list ports: %v", err)
	}

	if len(ports) != 2 {
		t.Fatalf("unexpected port count: got %d want 2 (%#v)", len(ports), ports)
	}
	if ports[0].Port != 3000 || ports[0].Host != "::1" || ports[0].Network != "tcp6" {
		t.Fatalf("unexpected tcp6 port: %#v", ports[0])
	}
	if ports[1].Port != 8080 || ports[1].Host != "127.0.0.1" || ports[1].Network != "tcp4" {
		t.Fatalf("unexpected tcp4 port: %#v", ports[1])
	}
}

func TestDecodeLocalAddressRejectsInvalidValue(t *testing.T) {
	t.Parallel()

	if _, _, err := decodeLocalAddress("invalid"); err == nil {
		t.Fatal("expected invalid local address to fail")
	}
}
