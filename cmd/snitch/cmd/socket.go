package cmd

import "github.com/fristovic/snitch/internal/ipc"

func resolveSocket() string {
	return ipc.ResolveSocket(socketPath)
}
