# Homebrew tap for Snitch

Goreleaser publishes the `snitch` formula here on each release tag.

## One-time setup (repo owner)

1. Create a public GitHub repo named `homebrew-snitch` under `fristovic`.
2. Add a GitHub Actions secret `HOMEBREW_TAP_TOKEN` on `fristovic/snitch` — a PAT with `repo` scope for `fristovic/homebrew-snitch`.
3. Tag a release (`v0.1.0`); goreleaser pushes `Formula/snitch.rb` automatically.

## Install (users)

```bash
brew tap fristovic/snitch
brew install snitch
open "$(brew --prefix)/opt/snitch/Snitch Bar.app"
snitch doctor
```

Snitch Bar manages the lie detector. Use **Start Snitching** / **Stop Snitching** in the menu bar — no `brew services` step.

## Uninstall

```bash
brew uninstall snitch
# or: snitch uninstall --purge
```

If you upgraded from an older install that used the legacy daemon LaunchAgent:

```bash
brew services stop snitch
```
