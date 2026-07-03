# Homebrew tap for Snitch

Goreleaser publishes the `snitch` formula here on each release tag.

## One-time setup (repo owner)

1. Create a public GitHub repo named `homebrew-snitch` under `fristovic`.

2. Create a **Personal Access Token** that can push to that repo:

   **Fine-grained (recommended)**

   - GitHub → Settings → Developer settings → Fine-grained tokens → Generate
   - Repository access: **Only select repositories** → `homebrew-snitch`
   - Permissions → Repository permissions → **Contents: Read and write**

   **Classic**

   - Scopes: `repo` (or at minimum access to `fristovic/homebrew-snitch`)

3. Add the token as a GitHub Actions secret on **`fristovic/snitch`** (not on the tap repo):

   - Settings → Secrets and variables → Actions → **New repository secret**
   - Name: `HOMEBREW_TAP_TOKEN`
   - Value: the PAT from step 2

4. Tag a release (`v0.1.2`, etc.); goreleaser pushes `Formula/snitch.rb` automatically.

If the secret is missing or expired, the GitHub release still publishes; the workflow logs a warning and skips the tap update until the token is fixed.

## Install (users)

```bash
brew tap fristovic/snitch
brew install snitch
snitch start
snitch doctor
```

Snitch Bar manages lie detection. Use **Start Snitching** / **Stop Snitching** in the menu bar — no `brew services` step.

## Uninstall

```bash
brew uninstall snitch
# or: snitch uninstall --purge
```

If you upgraded from an older install that used the legacy daemon LaunchAgent:

```bash
brew services stop snitch
```
