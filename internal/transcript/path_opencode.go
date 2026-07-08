package transcript

// OpenCodePathResolver is a no-op resolver: OpenCode stores sessions in SQLite,
// so there's no transcript file path to derive project/session metadata from.
// Project cwd comes from the session table (threaded through at poll time),
// and session ids come from the message rows. The reader supplies these
// directly when building TurnCompleted payloads.
type OpenCodePathResolver struct{}

func (OpenCodePathResolver) ProjectCwd(path string) string { return "" }
func (OpenCodePathResolver) ProjectDir(path string) string { return "" }
func (OpenCodePathResolver) SessionID(path string) string  { return "" }
