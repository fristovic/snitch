package transcript

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/event"
)

// TurnCompleted is emitted when a Cursor turn ends (turn_ended line).
type TurnCompleted struct {
	RunID          string     `json:"run_id"`
	SessionID      string     `json:"session_id"`
	TranscriptPath string     `json:"transcript_path"`
	ProjectPath    string     `json:"project_path"`
	StartHEAD      string     `json:"start_head,omitempty"`
	UserText       string     `json:"user_text"`
	AssistantText  string     `json:"assistant_text"`
	ToolCalls      []ToolCall `json:"tool_calls"`
	StartedAt      time.Time  `json:"started_at"`
	FinishedAt     time.Time  `json:"finished_at"`
	EndHEAD        string     `json:"end_head,omitempty"`
	FileManifest   map[string]string `json:"file_manifest,omitempty"`
}

// Watcher watches Cursor agent transcript JSONL files.
type Watcher struct {
	bus     *event.Bus
	cfg     config.CursorConfig
	root    string
	watcher *fsnotify.Watcher
	offsets map[string]int64
	turns   map[string]*turnBuffer
	watched map[string]bool
	mu      sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

type turnBuffer struct {
	userText      string
	assistantText strings.Builder
	toolCalls     []ToolCall
	toolResults   []ToolResult
	startedAt     time.Time
	startHEAD     string
	projectPath   string
}

// NewWatcher creates a Cursor transcript watcher.
func NewWatcher(bus *event.Bus, cfg config.CursorConfig) *Watcher {
	root := cfg.TranscriptWatchPath
	if root == "" {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, ".cursor", "projects")
	}
	return &Watcher{
		bus:     bus,
		cfg:     cfg,
		root:    root,
		offsets: make(map[string]int64),
		turns:   make(map[string]*turnBuffer),
		watched: make(map[string]bool),
	}
}

// Start begins watching transcript directories.
func (w *Watcher) Start() error {
	if !w.cfg.Enabled {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	w.ctx = ctx
	w.cancel = cancel

	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	w.watcher = fsw

	if err := w.walkAndWatch(w.root); err != nil && !os.IsNotExist(err) {
		_ = fsw.Close()
		return err
	}

	w.wg.Add(1)
	go w.loop()
	slog.Info("cursor watcher started", "root", w.root)
	return nil
}

// Stop shuts down the watcher.
func (w *Watcher) Stop() error {
	if w.cancel != nil {
		w.cancel()
	}
	if w.watcher != nil {
		_ = w.watcher.Close()
	}
	w.wg.Wait()
	return nil
}

// Name returns the watcher identifier.
func (w *Watcher) Name() string { return "cursor-transcript" }

func gitHEAD(projectPath string) string {
	if projectPath == "" {
		return ""
	}
	out, err := exec.Command("git", "-C", projectPath, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (w *Watcher) walkAndWatch(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			if w.owns(path) {
				w.seedOffsetEOF(path)
			}
			return nil
		}
		if strings.Contains(path, "agent-transcripts") || path == w.root {
			return w.watchDir(path)
		}
		return nil
	})
}

func (w *Watcher) watchDir(dir string) error {
	w.mu.Lock()
	if w.watched[dir] {
		w.mu.Unlock()
		return nil
	}
	w.watched[dir] = true
	w.mu.Unlock()
	if err := w.watcher.Add(dir); err != nil {
		return err
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.IsDir() {
			_ = w.watchDir(filepath.Join(dir, e.Name()))
		} else if w.owns(filepath.Join(dir, e.Name())) {
			w.seedOffsetEOF(filepath.Join(dir, e.Name()))
		}
	}
	return nil
}

func (w *Watcher) owns(path string) bool {
	return strings.Contains(path, "agent-transcripts") && strings.HasSuffix(path, ".jsonl")
}

func (w *Watcher) seedOffsetEOF(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}
	w.mu.Lock()
	w.offsets[path] = info.Size()
	w.mu.Unlock()
}

func (w *Watcher) loop() {
	defer w.wg.Done()
	for {
		select {
		case <-w.ctx.Done():
			return
		case ev, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(ev)
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			slog.Debug("fsnotify error", "err", err)
		}
	}
}

func (w *Watcher) handleEvent(ev fsnotify.Event) {
	if ev.Op&(fsnotify.Create|fsnotify.Write) == 0 {
		return
	}
	path := ev.Name
	if ev.Op&fsnotify.Create != 0 {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			_ = w.watchDir(path)
			return
		}
	}
	if !w.owns(path) {
		return
	}
	w.ingest(path)
}

func (w *Watcher) ingest(path string) {
	w.mu.Lock()
	off := w.offsets[path]
	w.mu.Unlock()

	lines, newOff, err := ParseLines(path, off)
	if err != nil {
		slog.Debug("parse transcript failed", "path", path, "err", err)
		return
	}
	w.mu.Lock()
	w.offsets[path] = newOff
	w.mu.Unlock()

	for _, line := range lines {
		w.handleLine(path, line)
	}
}

func (w *Watcher) handleLine(path string, line ParsedLine) {
	if line.TurnEnded {
		w.finishTurn(path)
		return
	}
	w.mu.Lock()
	buf := w.turns[path]
	if buf == nil {
		projectPath := ProjectCwdFromTranscriptPath(path)
		buf = &turnBuffer{
			startedAt:   time.Now(),
			startHEAD:   gitHEAD(projectPath),
			projectPath: projectPath,
		}
		w.turns[path] = buf
	}
	switch line.Role {
	case "user":
		if buf.userText == "" {
			buf.userText = line.Text
		} else if line.Text != "" {
			buf.userText += "\n" + line.Text
		}
	case "assistant":
		if line.Text != "" {
			if buf.assistantText.Len() > 0 {
				buf.assistantText.WriteString("\n")
			}
			buf.assistantText.WriteString(line.Text)
		}
		buf.toolCalls = append(buf.toolCalls, line.ToolCalls...)
		buf.toolResults = append(buf.toolResults, line.ToolResults...)
	default:
		buf.toolResults = append(buf.toolResults, line.ToolResults...)
	}
	w.mu.Unlock()
}

func (w *Watcher) finishTurn(path string) {
	w.mu.Lock()
	buf := w.turns[path]
	delete(w.turns, path)
	w.mu.Unlock()
	if buf == nil {
		return
	}
	if buf.userText == "" && buf.assistantText.Len() == 0 && len(buf.toolCalls) == 0 {
		return
	}

	finishedAt := time.Now()
	runID := uuid.NewString()
	sessionID := SessionIDFromTranscriptPath(path)
	projectPath := buf.projectPath
	if projectPath == "" {
		projectPath = ProjectCwdFromTranscriptPath(path)
	}

	payload := TurnCompleted{
		RunID:          runID,
		SessionID:      sessionID,
		TranscriptPath: path,
		ProjectPath:    projectPath,
		StartHEAD:      buf.startHEAD,
		UserText:       buf.userText,
		AssistantText:  buf.assistantText.String(),
		ToolCalls:      AttachToolResults(buf.toolCalls, buf.toolResults),
		StartedAt:      buf.startedAt,
		FinishedAt:     finishedAt,
		EndHEAD:        gitHEAD(projectPath),
		FileManifest:   BuildFileManifest(projectPath, AttachToolResults(buf.toolCalls, buf.toolResults)),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		slog.Warn("marshal TurnCompleted", "err", err)
		return
	}
	w.bus.Publish(event.Event{
		ID:        runID,
		Timestamp: finishedAt,
		Source:    "transcript",
		Type:      event.EventTurnCompleted,
		Payload:   data,
	})
	slog.Info("turn completed",
		"run_id", runID,
		"session_id", sessionID,
		"project", projectPath,
		"tool_calls", len(buf.toolCalls),
	)
}
