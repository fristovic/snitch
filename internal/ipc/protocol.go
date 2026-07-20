package ipc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/record"
	"github.com/fristovic/snitch/internal/transcript"
)

// Request is an IPC request.
type Request struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Response is an IPC response.
type Response struct {
	ID     string          `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *ErrorObj       `json:"error,omitempty"`
}

// ErrorObj is an IPC error.
type ErrorObj struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// EventMsg is a server-pushed event.
type EventMsg struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// Deps holds dependencies for the IPC server.
type Deps struct {
	Store      *record.Store
	Config     *config.Config
	ConfigPath string
	Version    string
	StartTime  time.Time
}

// Server handles IPC connections.
type Server struct {
	deps     Deps
	mu       sync.RWMutex
	subs     map[net.Conn]chan EventMsg
	listener net.Listener
}

// NewServer creates an IPC server.
func NewServer(deps Deps) *Server {
	if deps.StartTime.IsZero() {
		deps.StartTime = time.Now()
	}
	return &Server{deps: deps, subs: make(map[net.Conn]chan EventMsg)}
}

// Listen starts accepting connections.
func (s *Server) Listen() error {
	ln, err := listen(s.deps.Config.Daemon.SocketPath)
	if err != nil {
		return err
	}
	s.listener = ln
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go s.handle(conn)
		}
	}()
	return nil
}

// Close stops the server.
func (s *Server) Close() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// Broadcast sends an event to subscribers.
func (s *Server) Broadcast(event string, data any) {
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	msg := EventMsg{Event: event, Data: payload}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ch := range s.subs {
		select {
		case ch <- msg:
		default:
		}
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	defer s.unsubscribe(conn)
	scanner := bufio.NewScanner(conn)
	writer := bufio.NewWriter(conn)
	var writeMu sync.Mutex

	writeResp := func(resp Response) {
		data, _ := json.Marshal(resp)
		writeMu.Lock()
		_, _ = writer.Write(data)
		_ = writer.WriteByte('\n')
		_ = writer.Flush()
		writeMu.Unlock()
	}

	for scanner.Scan() {
		var req Request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			writeResp(Response{Error: &ErrorObj{Code: "PARSE_ERROR", Message: err.Error()}})
			continue
		}
		switch req.Method {
		case "status":
			s.handleStatus(req, writeResp)
		case "get_runs":
			s.handleGetRuns(req, writeResp)
		case "get_run":
			s.handleGetRun(req, writeResp)
		case "get_claims":
			s.handleGetClaims(req, writeResp)
		case "get_latest_top_claim":
			s.handleGetLatestTopClaim(req, writeResp)
		case "get_config":
			result, _ := json.Marshal(s.deps.Config)
			writeResp(Response{ID: req.ID, Result: result})
		case "set_config":
			s.handleSetConfig(req, writeResp)
		case "set_label":
			s.handleSetLabel(req, writeResp)
		case "add_missed_claim":
			s.handleAddMissedClaim(req, writeResp)
		case "subscribe":
			s.handleSubscribe(req, conn, writeResp, &writeMu, writer)
		default:
			writeResp(Response{ID: req.ID, Error: &ErrorObj{Code: "UNKNOWN_METHOD", Message: req.Method}})
		}
	}
}

func (s *Server) handleStatus(req Request, writeResp func(Response)) {
	total, _ := s.deps.Store.CountRuns()
	claimStats, _ := s.deps.Store.ClaimStats()
	projects, _ := s.deps.Store.CountDistinctProjects()
	sessions, _ := s.deps.Store.CountDistinctSessions()
	byHarness, _ := s.deps.Store.RunCountsByHarness()
	result, _ := json.Marshal(record.DaemonStatus{
		Running:                  true,
		UptimeSeconds:            int64(time.Since(s.deps.StartTime).Seconds()),
		Version:                  s.deps.Version,
		TotalRuns:                total,
		SnitchedRuns:             claimStats.SnitchedRuns,
		MostCommonFalseClaimType: claimStats.MostCommonFalseClaimType,
		TopClaimType:             claimStats.MostCommonFalseClaimType, // ≤0.3.x alias
		ProjectsWatched:          projects,
		SessionsSeen:             sessions,
		RunsByHarness:            byHarness,
		ClaimStats:               claimStats,
	})
	writeResp(Response{ID: req.ID, Result: result})
}

func (s *Server) handleGetRuns(req Request, writeResp func(Response)) {
	var p struct {
		Limit        int    `json:"limit"`
		Offset       int    `json:"offset"`
		Verdict      string `json:"verdict"`
		ProjectPath  string `json:"project_path"`
		SessionID    string `json:"session_id"`
		Harness      string `json:"harness"`
		Search       string `json:"search"`
		Since        string `json:"since"`
		FailuresOnly bool   `json:"failures_only"`
	}
	_ = json.Unmarshal(req.Params, &p)
	filter := record.RunFilter{
		Limit: p.Limit, Offset: p.Offset, Verdict: p.Verdict,
		ProjectPath: p.ProjectPath, SessionID: p.SessionID, Harness: p.Harness,
		Search: p.Search, FailuresOnly: p.FailuresOnly,
	}
	if p.Since != "" {
		if t, err := time.Parse(time.RFC3339, p.Since); err == nil {
			filter.Since = t
		}
	}
	runs, err := s.deps.Store.GetRuns(filter)
	if err != nil {
		writeResp(Response{ID: req.ID, Error: &ErrorObj{Code: "INTERNAL", Message: err.Error()}})
		return
	}
	result, _ := json.Marshal(runs)
	writeResp(Response{ID: req.ID, Result: result})
}

func (s *Server) handleGetRun(req Request, writeResp func(Response)) {
	var p struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(req.Params, &p)
	run, err := s.deps.Store.GetRunByID(p.ID)
	if err != nil || run == nil {
		writeResp(Response{ID: req.ID, Error: &ErrorObj{Code: "NOT_FOUND", Message: "run not found"}})
		return
	}
	claims, _ := s.deps.Store.GetClaimsByRun(run.ID)
	var toolCalls any
	if raw, err := s.deps.Store.GetRunPayloadJSON(run.ID); err == nil && len(raw) > 0 {
		var payload struct {
			ToolCalls []transcript.ToolCall `json:"tool_calls"`
		}
		if json.Unmarshal(raw, &payload) == nil && len(payload.ToolCalls) > 0 {
			toolCalls = payload.ToolCalls
		}
	}
	result, _ := json.Marshal(map[string]any{"run": run, "claims": claims, "tool_calls": toolCalls})
	writeResp(Response{ID: req.ID, Result: result})
}

func (s *Server) handleGetClaims(req Request, writeResp func(Response)) {
	var p struct {
		Limit           int    `json:"limit"`
		Offset          int    `json:"offset"`
		ClaimType       string `json:"claim_type"`
		ProjectPath     string `json:"project_path"`
		SessionID       string `json:"session_id"`
		Search          string `json:"search"`
		Since           string `json:"since"`
		FalseClaimsOnly bool   `json:"false_claims_only"`
		LiesOnly        bool   `json:"lies_only"` // deprecated ≤0.3.x alias
		MinSeverity     int    `json:"min_severity"`
	}
	_ = json.Unmarshal(req.Params, &p)
	filter := record.ClaimFilter{
		Limit: p.Limit, Offset: p.Offset, ClaimType: p.ClaimType,
		ProjectPath: p.ProjectPath, SessionID: p.SessionID, Search: p.Search,
		FalseClaimsOnly: p.FalseClaimsOnly || p.LiesOnly, MinSeverity: p.MinSeverity,
	}
	if p.Since != "" {
		if t, err := time.Parse(time.RFC3339, p.Since); err == nil {
			filter.Since = t
		}
	}
	claims, err := s.deps.Store.GetClaims(filter)
	if err != nil {
		writeResp(Response{ID: req.ID, Error: &ErrorObj{Code: "INTERNAL", Message: err.Error()}})
		return
	}
	result, _ := json.Marshal(claims)
	writeResp(Response{ID: req.ID, Result: result})
}

func (s *Server) handleGetLatestTopClaim(req Request, writeResp func(Response)) {
	claim, err := s.deps.Store.GetLatestTopFalseClaim()
	if err != nil {
		writeResp(Response{ID: req.ID, Error: &ErrorObj{Code: "INTERNAL", Message: err.Error()}})
		return
	}
	result, _ := json.Marshal(claim)
	writeResp(Response{ID: req.ID, Result: result})
}

func (s *Server) handleSetConfig(req Request, writeResp func(Response)) {
	var p struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	_ = json.Unmarshal(req.Params, &p)
	if err := s.deps.Config.Set(p.Key, p.Value); err != nil {
		writeResp(Response{ID: req.ID, Error: &ErrorObj{Code: "INVALID", Message: err.Error()}})
		return
	}
	if s.deps.ConfigPath != "" {
		if err := s.deps.Config.Save(s.deps.ConfigPath); err != nil {
			writeResp(Response{ID: req.ID, Error: &ErrorObj{Code: "INTERNAL", Message: err.Error()}})
			return
		}
	}
	result, _ := json.Marshal(map[string]bool{"ok": true})
	writeResp(Response{ID: req.ID, Result: result})
}

// handleSetLabel records a user's "Was this right?" verdict (correct/incorrect)
// on a run. shared reflects the user's opt-in to anonymous telemetry sharing.
func (s *Server) handleSetLabel(req Request, writeResp func(Response)) {
	var p struct {
		RunID   string `json:"run_id"`
		Label   string `json:"label"` // "correct" | "incorrect"
		Shared  *bool  `json:"shared"`
		Session string `json:"session"` // optional dedup key
	}
	_ = json.Unmarshal(req.Params, &p)
	if p.RunID == "" || (p.Label != "correct" && p.Label != "incorrect") {
		writeResp(Response{ID: req.ID, Error: &ErrorObj{Code: "INVALID", Message: "run_id and label (correct|incorrect) required"}})
		return
	}
	shared := s.deps.Config.Telemetry.ShareByDefault
	if p.Shared != nil {
		shared = *p.Shared
	}
	if err := s.deps.Store.SetRunLabel(p.RunID, p.Label, shared, p.Session); err != nil {
		writeResp(Response{ID: req.ID, Error: &ErrorObj{Code: "INTERNAL", Message: err.Error()}})
		return
	}
	result, _ := json.Marshal(map[string]any{"ok": true, "shared": shared})
	writeResp(Response{ID: req.ID, Result: result})
}

// handleAddMissedClaim records a user-reported false negative (the agent lied
// about something Snitch missed). Claim/actual text stays local; only metadata
// is ever shared via telemetry.
func (s *Server) handleAddMissedClaim(req Request, writeResp func(Response)) {
	var p struct {
		RunID   string `json:"run_id"`
		Claimed string `json:"claimed"`
		Actual  string `json:"actual"`
		Shared  *bool  `json:"shared"`
	}
	_ = json.Unmarshal(req.Params, &p)
	if p.Claimed == "" || p.Actual == "" {
		writeResp(Response{ID: req.ID, Error: &ErrorObj{Code: "INVALID", Message: "claimed and actual required"}})
		return
	}
	shared := s.deps.Config.Telemetry.ShareByDefault
	if p.Shared != nil {
		shared = *p.Shared
	}
	if err := s.deps.Store.AddMissedClaim(p.RunID, p.Claimed, p.Actual, shared); err != nil {
		writeResp(Response{ID: req.ID, Error: &ErrorObj{Code: "INTERNAL", Message: err.Error()}})
		return
	}
	result, _ := json.Marshal(map[string]any{"ok": true, "shared": shared})
	writeResp(Response{ID: req.ID, Result: result})
}

func (s *Server) handleSubscribe(req Request, conn net.Conn, writeResp func(Response), writeMu *sync.Mutex, writer *bufio.Writer) {
	ch := make(chan EventMsg, 32)
	s.mu.Lock()
	if old, ok := s.subs[conn]; ok {
		delete(s.subs, conn)
		close(old)
	}
	s.subs[conn] = ch
	s.mu.Unlock()
	go func() {
		for msg := range ch {
			data, _ := json.Marshal(msg)
			writeMu.Lock()
			_, _ = writer.Write(data)
			_ = writer.WriteByte('\n')
			_ = writer.Flush()
			writeMu.Unlock()
		}
	}()
	writeResp(Response{ID: req.ID, Result: json.RawMessage(`{"subscribed":true}`)})
}

func (s *Server) unsubscribe(conn net.Conn) {
	s.mu.Lock()
	ch, ok := s.subs[conn]
	if ok {
		delete(s.subs, conn)
	}
	s.mu.Unlock()
	if ok {
		close(ch)
	}
}

// Client communicates with snitchd.
type Client struct {
	conn net.Conn
	mu   sync.Mutex
	id   int
}

// Connect dials the daemon.
func Connect(addr string) (*Client, error) {
	conn, err := dial(addr)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn}, nil
}

// Close closes the connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// Call invokes an IPC method.
func (c *Client) Call(method string, params any) (json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.id++
	req := Request{ID: formatID(c.id), Method: method}
	if params != nil {
		req.Params, _ = json.Marshal(params)
	}
	data, _ := json.Marshal(req)
	if _, err := c.conn.Write(append(data, '\n')); err != nil {
		return nil, err
	}
	scanner := newScanner(c.conn)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}
		return nil, errors.New("no response")
	}
	var resp Response
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, errors.New(resp.Error.Message)
	}
	return resp.Result, nil
}

func formatID(n int) string {
	return "req-" + strconv.Itoa(n)
}

// Watch subscribes to daemon events and invokes handler for each pushed message until ctx is cancelled.
func Watch(ctx context.Context, addr string, handler func(EventMsg) error) error {
	conn, err := dial(addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	req := Request{ID: "watch-1", Method: "subscribe"}
	data, _ := json.Marshal(req)
	if _, err := conn.Write(append(data, '\n')); err != nil {
		return err
	}

	scanner := newScanner(conn)
	if !scanner.Scan() {
		return errors.New("no subscribe response")
	}
	var resp Response
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		return err
	}
	if resp.Error != nil {
		return errors.New(resp.Error.Message)
	}

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	for scanner.Scan() {
		var msg EventMsg
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		if err := handler(msg); err != nil {
			return err
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return scanner.Err()
}

// newScanner returns a line scanner with an 8MiB max token size for large IPC payloads.
func newScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	return scanner
}
