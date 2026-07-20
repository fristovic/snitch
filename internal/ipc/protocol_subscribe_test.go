package ipc

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/record"
)

func ipcTestDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "sn")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

func TestSubscribeUnsubscribeOnDisconnect(t *testing.T) {
	dir := ipcTestDir(t)
	store, err := record.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	sock := filepath.Join(dir, "ipc.sock")
	cfg := config.Default()
	cfg.Daemon.SocketPath = sock
	srv := NewServer(Deps{Store: store, Config: cfg, Version: "test"})
	if err := srv.Listen(); err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	client, err := Connect(sock)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Call("subscribe", nil); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)

	srv.mu.RLock()
	n := len(srv.subs)
	srv.mu.RUnlock()
	if n != 1 {
		t.Fatalf("expected 1 subscriber, got %d", n)
	}

	_ = client.Close()
	time.Sleep(50 * time.Millisecond)

	srv.mu.RLock()
	n = len(srv.subs)
	srv.mu.RUnlock()
	if n != 0 {
		t.Fatalf("expected 0 subscribers after disconnect, got %d", n)
	}
}

func TestSetLabelSharedExplicitFalse(t *testing.T) {
	dir := ipcTestDir(t)
	store, err := record.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	_ = store.InsertRun(record.Run{ID: "r1", DeviceID: "d"})

	sock := filepath.Join(dir, "ipc.sock")
	cfg := config.Default()
	cfg.Daemon.SocketPath = sock
	cfg.Telemetry.ShareByDefault = true
	srv := NewServer(Deps{Store: store, Config: cfg, Version: "test"})
	if err := srv.Listen(); err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	client, err := Connect(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	falseVal := false
	_, err = client.Call("set_label", map[string]any{
		"run_id": "r1",
		"label":  "correct",
		"shared": falseVal,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, shared, _, _, err := store.GetRunLabel("r1")
	if err != nil {
		t.Fatal(err)
	}
	if shared {
		t.Fatal("expected shared=false to override share_by_default")
	}
}
