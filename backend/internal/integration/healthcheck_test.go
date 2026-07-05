package integration

import (
	"net"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestHealthCheckFlag builds the real binary and exercises the -health-check
// flag exactly as the docker-compose healthcheck does: exit 0 against a
// serving instance, non-zero against a dead port.
func TestHealthCheckFlag(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain not available for exec test")
	}

	bin := filepath.Join(t.TempDir(), "server")
	build := exec.Command("go", "build", "-o", bin, "../../cmd/server")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	h := Start(t, nil, nil)
	_, port, err := net.SplitHostPort(strings.TrimPrefix(h.BaseURL, "http://"))
	if err != nil {
		t.Fatal(err)
	}

	probe := exec.Command(bin, "-health-check")
	probe.Env = append(probe.Environ(), "PORT="+port)
	if out, err := probe.CombinedOutput(); err != nil {
		t.Fatalf("health check against live server failed: %v\n%s", err, out)
	}

	dead := exec.Command(bin, "-health-check")
	dead.Env = append(dead.Environ(), "PORT=1") // nothing listens there
	if err := dead.Run(); err == nil {
		t.Fatal("health check against a dead port must exit non-zero")
	}
}
