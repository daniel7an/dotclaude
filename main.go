package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "--version", "-v":
		fmt.Println("dotclaude " + version)
	case "--help", "-h", "help":
		printUsage()
	case "init":
		if len(os.Args) < 3 {
			fatal("Usage: dotclaude init <repo-url>")
		}
		cmdInit(os.Args[2])
	case "push":
		cmdPush()
	case "pull":
		cmdPull()
	case "status":
		cmdStatus()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage: dotclaude <command>

Commands:
  init <repo-url>   Clone config repo to ~/.dotclaude/repo/
  push              Sync local ~/.claude/ config to repo and push
  pull              Pull from repo and restore config to ~/.claude/
  status            Show differences between local and repo`)
}

func cmdInit(repoURL string) {
	dotclaude := dotclaudeDir()
	repoPath := filepath.Join(dotclaude, "repo")

	if dirExists(repoPath) {
		// Check if it's already a valid repo
		if dirExists(filepath.Join(repoPath, ".git")) {
			fmt.Println("Repo already initialized at", repoPath)
			fmt.Println("Run 'dotclaude push' to save or 'dotclaude pull' to restore.")
			return
		}
		// Remove invalid repo dir and re-clone
		os.RemoveAll(repoPath)
	}

	if err := os.MkdirAll(dotclaude, 0755); err != nil {
		fatal("Failed to create ~/.dotclaude: %v", err)
	}

	fmt.Printf("Cloning %s...\n", repoURL)
	if err := gitClone(repoURL, repoPath); err != nil {
		fatal("%v", err)
	}

	fmt.Println("Ready. Run 'dotclaude push' to save your config.")
}

func cmdPush() {
	repoPath := requireRepo()
	claudeDir := claudeHomeDir()

	pairs := collectPushFiles(claudeDir, repoPath)
	if len(pairs) == 0 {
		fmt.Println("No files to sync.")
		return
	}

	// Copy files to repo
	for _, p := range pairs {
		if err := copyFile(p.Src, p.Dst); err != nil {
			fatal("Failed to copy %s: %v", p.RelPath, err)
		}
	}

	// Clean up repo files that no longer exist locally
	cleanupDeletedFiles(claudeDir, repoPath, pairs)

	// Git add, commit, push
	if err := gitAdd(repoPath); err != nil {
		fatal("%v", err)
	}

	if !gitHasChanges(repoPath) {
		fmt.Println("Nothing changed. Already up to date.")
		return
	}

	if err := gitCommit(repoPath, "dotclaude: sync config"); err != nil {
		fatal("%v", err)
	}

	if err := gitPush(repoPath); err != nil {
		fatal("%v", err)
	}

	fmt.Printf("Synced %d files to repo.\n", len(pairs))
}

func cmdPull() {
	repoPath := requireRepo()
	claudeDir := claudeHomeDir()
	backupBase := filepath.Join(dotclaudeDir(), "backups")

	// Git pull
	if err := gitPull(repoPath); err != nil {
		fatal("%v", err)
	}

	pairs := collectPullFiles(claudeDir, repoPath)
	if len(pairs) == 0 {
		fmt.Println("No files in repo to restore.")
		return
	}

	// Backup existing files before overwriting
	backupDir, backupCount, err := backupFiles(pairs, backupBase)
	if err != nil {
		fatal("Backup failed: %v", err)
	}

	// Copy files from repo to ~/.claude/
	for _, p := range pairs {
		if err := copyFile(p.Src, p.Dst); err != nil {
			fatal("Failed to restore %s: %v", p.RelPath, err)
		}
	}

	// Prune old backups
	pruneBackups(backupBase, maxBackups)

	fmt.Printf("Applied %d files.", len(pairs))
	if backupCount > 0 {
		fmt.Printf(" Backup at %s", backupDir)
	}
	fmt.Println()
}

func cmdStatus() {
	repoPath := requireRepo()
	claudeDir := claudeHomeDir()

	localFiles := collectLocalFiles(claudeDir)
	repoFiles := collectRepoFiles(repoPath)

	// Collect all keys
	allKeys := make(map[string]bool)
	for k := range localFiles {
		allKeys[k] = true
	}
	for k := range repoFiles {
		allKeys[k] = true
	}

	var modified, newLocal, newRepo []string

	sorted := sortedKeys(allKeys)
	for _, key := range sorted {
		localSum, inLocal := localFiles[key]
		repoSum, inRepo := repoFiles[key]

		switch {
		case inLocal && inRepo && localSum != repoSum:
			modified = append(modified, key)
		case inLocal && !inRepo:
			newLocal = append(newLocal, key)
		case !inLocal && inRepo:
			newRepo = append(newRepo, key)
		}
	}

	if len(modified) == 0 && len(newLocal) == 0 && len(newRepo) == 0 {
		fmt.Println("Everything in sync.")
		return
	}

	for _, f := range modified {
		fmt.Printf("  Modified: %s\n", f)
	}
	for _, f := range newLocal {
		fmt.Printf("  New locally: %s\n", f)
	}
	for _, f := range newRepo {
		fmt.Printf("  New in repo: %s\n", f)
	}
}

// cleanupDeletedFiles removes files from the repo that are no longer present locally.
func cleanupDeletedFiles(claudeDir, repoDir string, currentPairs []FilePair) {
	existing := collectRepoFiles(repoDir)
	currentSet := make(map[string]bool)
	for _, p := range currentPairs {
		currentSet[p.RelPath] = true
	}

	for rel := range existing {
		if !currentSet[rel] {
			os.Remove(filepath.Join(repoDir, rel))
		}
	}
}

// Helper functions

func dotclaudeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fatal("Cannot determine home directory: %v", err)
	}
	return filepath.Join(home, ".dotclaude")
}

func claudeHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fatal("Cannot determine home directory: %v", err)
	}
	return filepath.Join(home, ".claude")
}

func requireRepo() string {
	repoPath := filepath.Join(dotclaudeDir(), "repo")
	if !dirExists(filepath.Join(repoPath, ".git")) {
		fatal("No repo found. Run 'dotclaude init <repo-url>' first.")
	}
	return repoPath
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
