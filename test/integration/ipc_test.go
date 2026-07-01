package integration_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/ipc"
	"github.com/fristovic/snitch/internal/record"
)

func ipcTestSocket(t *testing.T, name string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		return `\\.\pipe\snitch-test-` + name
	}
	return filepath.Join(t.TempDir(), name+".sock")
}

func TestIPCStatusAndLieStats(t *testing.T) {
	dir := t.TempDir()
	store, err := record.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	_ = store.InsertRun(record.Run{ID: "r1", Verdict: record.VerdictFail, DeviceID: "d"})
	_ = store.InsertClaims([]record.Claim{{
		RunID: "r1", ClaimType: "test_pass", Source: "prose",
		Claimed: "all tests pass", Actual: "no tests", Verified: -1, Severity: 3,
	}})

	sock := ipcTestSocket(t, "status")
	cfg := config.Default()
	cfg.Daemon.SocketPath = sock

	srv := ipc.NewServer(ipc.Deps{
		Store:   store,
		Config:  cfg,
		Version: "test",
	})
	if err := srv.Listen(); err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	client, err := ipc.Connect(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	stData, err := client.Call("status", nil)
	if err != nil {
		t.Fatal(err)
	}
	var st record.DaemonStatus
	if err := json.Unmarshal(stData, &st); err != nil {
		t.Fatal(err)
	}
	if st.SnitchedRuns != 1 {
		t.Fatalf("expected snitched=1, got %d", st.SnitchedRuns)
	}
	if st.LieStats.ByClaimType["test_pass"] != 1 {
		t.Fatalf("expected test_pass lie, got %+v", st.LieStats.ByClaimType)
	}

	claimsData, err := client.Call("get_claims", map[string]any{"lies_only": true})
	if err != nil {
		t.Fatal(err)
	}
	var claims []record.LieClaim
	if err := json.Unmarshal(claimsData, &claims); err != nil {
		t.Fatal(err)
	}
	if len(claims) != 1 || claims[0].ClaimType != "test_pass" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestIPCWatchBroadcast(t *testing.T) {
	store, _ := record.Open(t.TempDir())
	defer store.Close()

	sock := ipcTestSocket(t, "watch")
	cfg := config.Default()
	cfg.Daemon.SocketPath = sock
	srv := ipc.NewServer(ipc.Deps{Store: store, Config: cfg, Version: "test"})
	if err := srv.Listen(); err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	received := make(chan struct{}, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	go func() {
		_ = ipc.Watch(ctx, sock, func(msg ipc.EventMsg) error {
			if msg.Event == "run.completed" {
				select {
				case received <- struct{}{}:
				default:
				}
				cancel()
			}
			return nil
		})
	}()

	time.Sleep(50 * time.Millisecond)
	srv.Broadcast("run.completed", map[string]string{"run_id": "abc", "verdict": "pass"})

	select {
	case <-received:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for broadcast")
	}
}
