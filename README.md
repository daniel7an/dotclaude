# dotclaude

Sync your [Claude Code](https://docs.anthropic.com/en/docs/claude-code) configuration across machines using a private Git repo.

## What it syncs

- `settings.json`, `settings.local.json`
- `CLAUDE.md`
- `keybindings.json`
- Skills (`skills/*/SKILL.md`)
- Plugins and marketplace configs

Credentials and caches are never synced.

## Install

### Homebrew

```
brew install daniel7an/tap/dotclaude
```

### From source

```
go install github.com/daniel7an/dotclaude@latest
```

## Usage

```bash
# Initialize with a private repo
dotclaude init git@github.com:you/claude-config.git

# Push local config to repo
dotclaude push

# Pull config from repo to local
dotclaude pull

# Show differences
dotclaude status
```

## How it works

`dotclaude push` copies files from `~/.claude/` into a local clone at `~/.dotclaude/repo/`, commits, and pushes. `dotclaude pull` does the reverse — pulls from the repo and restores files to `~/.claude/`. JSON files are merged intelligently; non-JSON files use a last-write-wins strategy. Backups are created before any pull overwrites existing files.

## License

MIT
