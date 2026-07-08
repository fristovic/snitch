# v1.0.0 Launch Checklist (GitHub-first)

v1 launches as a GitHub project. snitchworks.com hosts only the placeholder in
`server/site/` plus the telemetry endpoints in `server/telemetry/`. The full
landing page is a v3 (Snitchworks) deliverable.

## Before tagging

- [ ] **Dogfood week** — run `snitch replay ~/.cursor/projects` (and `~/.claude/projects`,
      `~/.codex/sessions`, `~/.pi/agent/sessions` with `--harness`) over your real
      session history. Review every `LIE` line; each false positive is a pattern
      to fix. Target: a false-positive rate you can state publicly.
- [ ] **Live validation per harness** — run the daemon with each harness enabled
      while doing a real session; confirm turns appear in `snitch status --detailed`
      under `runs by harness`.
- [ ] **Demo GIF** — screen-record Snitch catching a live Claude Code lie
      (menu bar alert → Show Last Lie). Replace the TODO placeholder at the top
      of README.md. This doubles as the live-validation proof.
- [ ] **Clean-machine install** — `brew tap && brew install && snitch start` on a
      machine (or fresh user account) that has never seen Snitch.
- [ ] **Deploy telemetry** — `server/telemetry/README.md`; verify
      `snitch config set telemetry.enabled true` + a labeled run round-trips into
      the `labels` table.
- [ ] **Deploy placeholder** — `server/site/index.html` at snitchworks.com
      (any static host; it's a single file).

## Tag + release

- [ ] `git tag v1.0.0 && git push --tags` — goreleaser publishes binaries + Homebrew tap.
- [ ] Verify `brew upgrade snitch` picks up v1.0.0.

## Launch posts

- [ ] Show HN: "Show HN: Snitch — a lie detector for your AI coding agent"
- [ ] Reddit: r/programming, r/ClaudeCode (as appropriate), r/cursor
- [ ] Product Hunt (Tue–Thu)
- [ ] X/Twitter personal account

## Post-launch (first 48h)

- [ ] Triage GitHub issues; v1.0.1 bugfix within 48h if needed.
- [ ] Watch telemetry opt-in rate (playbook target: 15–30% of active users).
