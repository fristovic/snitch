# 0.3.1 Release Checklist (GitHub-first)

0.3.1 ships multi-harness ingestion, the data flywheel, and Snitch Bar–owned
notifications. snitchworks.com hosts only the placeholder in `server/site/`
plus the telemetry endpoints in `server/telemetry/`. A full marketing site is
a later Snitchworks deliverable.

## Before tagging

- [ ] **Dogfood** — run `snitch replay` over real Cursor (and optionally other
      harness) history; note false-positive rate you can state publicly.
- [ ] **Live Cursor smoke** — one synthetic lie → menu alert + Notification
      Center with Snitch AppIcon → **View Details…** opens `snitch log`.
- [ ] **Clean-machine install** — `brew tap && brew install && snitch start`
      (or curl install) on a machine/account that is not the e2e Cellar hot-swap.
- [ ] **Optional:** enable Claude/Codex/Pi/OpenCode one at a time; confirm
      `snitch status --detailed` shows runs by harness.
- [ ] **Optional for flywheel:** deploy telemetry (`server/telemetry/README.md`)
      and verify `telemetry.enabled` + a labeled run round-trips.
- [ ] **Optional:** deploy `server/site/index.html` at snitchworks.com.

## Tag + release

- [ ] Push `main` (docs + changelog for 0.3.1).
- [ ] `git tag v0.3.1 && git push origin v0.3.1` — goreleaser publishes
      binaries + Homebrew tap.
- [ ] Verify GitHub Release assets include `Snitch Bar.app` and `AppIcon.icns`.
- [ ] Verify `brew upgrade snitch` (or fresh install) reports version `0.3.1`
      and `snitch doctor` is green.

## Announce (after brew is live)

- [ ] Show HN / Reddit / Product Hunt / X as desired.
- [ ] Triage issues; patch release if needed within 48h.
