//go:build darwin

package main

import (
	"log/slog"
	"os"

	"github.com/fristovic/snitch/internal/ipc"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})))

	socket := ipc.ResolveSocket(os.Getenv("SNITCH_SOCKET"))
	if socket == "" {
		slog.Error("could not resolve IPC socket")
		os.Exit(1)
	}

	app := newTrayApp(socket)
	app.run()
}
