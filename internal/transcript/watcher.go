package transcript

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"

	"github.com/fristovic/snitch/internal/event"
)

// TurnCompleted is emitted when an agent turn ends. Harness names the source
// platform ("cursor", "claude", "codex", "pi", "opencode").
type TurnCompleted struct {
	RunID          string            `json:"run_id"`
	SessionID      string            `json:"session_id"`
	TranscriptPath string            `json:"transcript_path"`
	ProjectPath    string            `json:"project_path"`
	Harness        string            `json:"harness,omitempty"`
	Model          string            `json:"model,omitempty"`
	StartHEAD      string            `json:"start_head,omitempty"`
	UserText       string            `json:"user_text"`
	AssistantText  string            `json:"assistant_text"`
	ToolCalls      []ToolCall        `json:"tool_calls"`
	StartedAt      time.Time         `json:"started_at"`
	FinishedAt     time.Time         `json:"finished_at"`
	EndHEAD        string            `json:"end_head,omitempty"`
	FileManifest   map[string]string `json:"file_manifest,omitempty"`
}

// defaultIdleFlush is how long a turn buffer may sit without new writes
// before the watcher flushes it. Pi and Codex only mark turn boundaries on
// the NEXT user message / turn_context, so a session's final turn would
// otherwise be lost.
const defaultIdleFlush = 30 * time.Second

// Watcher watches agent transcript JSONL files for one harness. It owns
// fsnotify plumbing, file offsets, and goroutine lifecycle; turn-boundary
// semantics live in turnAssembler.
type Watcher struct {
	bus        *event.Bus
	cfg        WatcherConfig
	watcher    *fsnotify.Watcher
	offsets    map[string]int64
	assemblers map[string]*turnAssembler
	watched    map[string]bool
	mu         sync.Mutex
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// NewWatcherWith creates a harness-agnostic transcript watcher.
// One per enabled JSONL harness; the daemon starts all of them.
func NewWatcherWith(bus *event.Bus, cfg WatcherConfig) *Watcher {
	if cfg.IdleFlush <= 0 {
		cfg.IdleFlush = defaultIdleFlush
	}
	return &Watcher{
		bus:        bus,
		cfg:        cfg,
		offsets:    make(map[string]int64),
		assemblers: make(map[string]*turnAssembler),
		watched:    make(map[string]bool),
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

	if err := w.walkAndWatch(w.cfg.Root); err != nil && !os.IsNotExist(err) {
		_ = fsw.Close()
		return err
	}

	w.wg.Add(1)
	go w.loop()
	slog.Info("transcript watcher started", "harness", w.cfg.Harness, "root", w.cfg.Root)
	return nil
}

// Stop shuts down the watcher, draining any in-flight turn buffers so the
// session's final turn is not lost.
func (w *Watcher) Stop() error {
	if w.cancel != nil {
		w.cancel()
	}
	if w.watcher != nil {
		_ = w.watcher.Close()
	}
	w.wg.Wait()

	w.mu.Lock()
	pending := make(map[string]*turnBuffer)
	for path, a := range w.assemblers {
		if buf := a.Drain(); buf != nil {
			pending[path] = buf
		}
	}
	w.mu.Unlock()
	for path, buf := range pending {
		w.emitTurn(path, buf)
	}
	return nil
}

func (w *Watcher) walkAndWatch(root string) error {
	ownsDir := w.cfg.OwnsDir
	if ownsDir == nil {
		ownsDir = func(string) bool { return true }
	}
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
		if ownsDir(path) || path == w.cfg.Root {
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
	if w.cfg.OwnsFile != nil {
		return w.cfg.OwnsFile(path)
	}
	return strings.HasSuffix(path, ".jsonl")
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
	idle := time.NewTicker(w.cfg.IdleFlush / 2)
	defer idle.Stop()
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-idle.C:
			w.flushIdle()
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

// flushIdle emits any turn buffer that has not received writes recently.
// This covers harnesses (Pi, Codex) whose final turn has no explicit end
// marker until the next user message, which never arrives at session end.
func (w *Watcher) flushIdle() {
	cutoff := time.Now().Add(-w.cfg.IdleFlush)
	w.mu.Lock()
	pending := make(map[string]*turnBuffer)
	for path, a := range w.assemblers {
		if buf := a.Idle(cutoff); buf != nil {
			pending[path] = buf
		}
	}
	w.mu.Unlock()
	for path, buf := range pending {
		w.emitTurn(path, buf)
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

	lines, newOff, err := ParseLinesWith(w.cfg.Parser, path, off)
	if err != nil {
		slog.Debug("parse transcript failed", "path", path, "err", err)
		return
	}
	w.mu.Lock()
	w.offsets[path] = newOff
	a := w.assemblers[path]
	if a == nil {
		a = newTurnAssembler(w.cfg.Resolver, path)
		w.assemblers[path] = a
	}
	var completed []*turnBuffer
	for _, line := range lines {
		if buf := a.Feed(line); buf != nil {
			completed = append(completed, buf)
		}
	}
	w.mu.Unlock()

	for _, buf := range completed {
		w.emitTurn(path, buf)
	}
}

// emitTurn converts a completed buffer into a TurnCompleted event.
func (w *Watcher) emitTurn(path string, buf *turnBuffer) {
	projectPath := buf.projectPath
	if projectPath == "" {
		projectPath = w.cfg.Resolver.ProjectCwd(path)
	}
	toolCalls := AttachToolResults(buf.toolCalls, buf.toolResults)
	PublishTurnCompleted(w.bus, TurnCompleted{
		RunID:          uuid.NewString(),
		SessionID:      w.cfg.Resolver.SessionID(path),
		TranscriptPath: path,
		ProjectPath:    projectPath,
		Harness:        w.cfg.Harness,
		Model:          buf.model,
		StartHEAD:      buf.startHEAD,
		UserText:       buf.userText,
		AssistantText:  buf.assistantText.String(),
		ToolCalls:      toolCalls,
		StartedAt:      buf.startedAt,
		FinishedAt:     time.Now(),
		EndHEAD:        GitHEAD(projectPath),
		FileManifest:   BuildFileManifest(projectPath, toolCalls),
	})
}
