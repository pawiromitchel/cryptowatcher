//go:build !windows

package e2e

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"
)

var (
	buildOnce sync.Once
	binPath   string
	buildErr  error
)

// binary compiles the real cmd/cryptowatcher once per test run.
func binary(t *testing.T) string {
	t.Helper()
	buildOnce.Do(func() {
		dir, err := os.MkdirTemp("", "cryptowatcher-e2e-*")
		if err != nil {
			buildErr = err
			return
		}
		binPath = filepath.Join(dir, "cryptowatcher")
		cmd := exec.Command("go", "build", "-ldflags", "-X main.version=v9.9.9-test", "-o", binPath, "../cmd/cryptowatcher")
		if out, err := cmd.CombinedOutput(); err != nil {
			buildErr = &buildError{err, string(out)}
		}
	})
	if buildErr != nil {
		t.Fatalf("building binary: %v", buildErr)
	}
	return binPath
}

type buildError struct {
	err error
	out string
}

func (b *buildError) Error() string { return b.err.Error() + "\n" + b.out }

func run(t *testing.T, xdg string, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(binary(t), args...)
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+xdg)
	var so, se bytes.Buffer
	cmd.Stdout, cmd.Stderr = &so, &se
	err := cmd.Run()
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return so.String(), se.String(), code
}

func TestBinaryVersionAndConfigPath(t *testing.T) {
	xdg := t.TempDir()

	out, _, code := run(t, xdg, "-version")
	if code != 0 || strings.TrimSpace(out) != "cryptowatcher v9.9.9-test" {
		t.Errorf("-version: code=%d out=%q", code, out)
	}

	out, _, code = run(t, xdg, "-config-path")
	want := filepath.Join(xdg, "cryptowatcher", "config.json")
	if code != 0 || strings.TrimSpace(out) != want {
		t.Errorf("-config-path: code=%d out=%q want %q", code, out, want)
	}
}

func TestBinaryRejectsBadFlags(t *testing.T) {
	xdg := t.TempDir()

	_, stderr, code := run(t, xdg, "-interval", "1")
	if code != 2 || !strings.Contains(stderr, "at least 5") {
		t.Errorf("-interval 1: code=%d stderr=%q", code, stderr)
	}
	if _, _, code = run(t, xdg, "-nonsense"); code == 0 {
		t.Error("unknown flags must fail")
	}
}

// term is a running binary attached to a pseudo-terminal.
type term struct {
	t    *testing.T
	ptmx *os.File
	cmd  *exec.Cmd
	mu   sync.Mutex
	buf  bytes.Buffer
}

func startTerm(t *testing.T, xdg string, args ...string) *term {
	t.Helper()
	cmd := exec.Command(binary(t), args...)
	cmd.Env = append(os.Environ(), "XDG_CONFIG_HOME="+xdg, "TERM=xterm-256color")
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 45, Cols: 140})
	if err != nil {
		t.Skipf("cannot allocate a pty here: %v", err)
	}
	tm := &term{t: t, ptmx: ptmx, cmd: cmd}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = ptmx.Close() })
	go func() {
		chunk := make([]byte, 4096)
		for {
			n, err := ptmx.Read(chunk)
			tm.mu.Lock()
			tm.buf.Write(chunk[:n])
			tm.mu.Unlock()
			if err != nil {
				return
			}
		}
	}()
	return tm
}

func (tm *term) seen() string { tm.mu.Lock(); defer tm.mu.Unlock(); return tm.buf.String() }

func (tm *term) waitFor(sub string) {
	tm.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(tm.seen(), sub) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	tm.t.Fatalf("timed out waiting for %q; output so far:\n%q", sub, tm.seen())
}

func (tm *term) send(s string) { _, _ = io.WriteString(tm.ptmx, s) }

func (tm *term) wantCleanExit() {
	tm.t.Helper()
	done := make(chan error, 1)
	go func() { done <- tm.cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			tm.t.Fatalf("binary exited uncleanly: %v", err)
		}
	case <-time.After(5 * time.Second):
		tm.t.Fatal("binary did not exit")
	}
}

// TestBinaryRunsInRealTerminal launches the real executable on a pseudo-terminal
// in -mock mode, checks the dashboard draws, navigates, and quits cleanly.
func TestBinaryRunsInRealTerminal(t *testing.T) {
	xdg := t.TempDir()
	tm := startTerm(t, xdg, "-mock")

	tm.waitFor("CRYPTOWATCHER")
	tm.waitFor("BTC/USD")
	tm.waitFor("LIVE") // mock data resolves immediately

	tm.send("?") // expand help
	tm.waitFor("move earlier")
	tm.send("\r") // open details
	tm.waitFor("Source")
	tm.send("\x1b") // esc closes details
	time.Sleep(200 * time.Millisecond)
	tm.send("q")
	tm.wantCleanExit()

	if !fileExists(filepath.Join(xdg, "cryptowatcher", "config.json")) {
		t.Error("the binary should create its default config on first run")
	}
}

// TestBinaryWarnsAboutCorruptConfigWithoutLosingIt starts the binary with a
// broken config file: it must still run (with defaults), tell the user, and
// keep the original file as config.json.bak.
func TestBinaryWarnsAboutCorruptConfigWithoutLosingIt(t *testing.T) {
	xdg := t.TempDir()
	cfgDir := filepath.Join(xdg, "cryptowatcher")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	const junk = `{"crypto_pairs": ["BTC-USD",`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(junk), 0o644); err != nil {
		t.Fatal(err)
	}

	tm := startTerm(t, xdg, "-mock")
	tm.waitFor("Config warning")
	tm.waitFor("BTC/USD")
	tm.send("q")
	tm.wantCleanExit()

	backup, err := os.ReadFile(filepath.Join(cfgDir, "config.json.bak"))
	if err != nil || string(backup) != junk {
		t.Errorf("original config must be preserved as .bak: %q, %v", backup, err)
	}
}
