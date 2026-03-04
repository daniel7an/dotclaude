package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Files to sync (relative to ~/.claude/)
var syncFiles = []string{
	"settings.json",
	"settings.local.json",
	"CLAUDE.md",
	"plugins/installed_plugins.json",
	"plugins/blocklist.json",
	"plugins/known_marketplaces.json",
}

// Glob patterns to sync
var syncGlobs = []string{}

// Never sync these (defense in depth)
var neverSync = []string{
	".credentials.json",
}

// FilePair maps a source file to its destination
type FilePair struct {
	Src     string // absolute path
	Dst     string // absolute path
	RelPath string // relative path within repo
}

// collectPushFiles gathers files from ~/.claude/ for pushing to the repo
func collectPushFiles(claudeDir, repoDir string) []FilePair {
	var pairs []FilePair

	// Static files
	for _, rel := range syncFiles {
		src := filepath.Join(claudeDir, rel)
		if fileExists(src) {
			pairs = append(pairs, FilePair{
				Src:     src,
				Dst:     filepath.Join(repoDir, rel),
				RelPath: rel,
			})
		}
	}

	// Glob patterns
	for _, pattern := range syncGlobs {
		matches, _ := filepath.Glob(filepath.Join(claudeDir, pattern))
		for _, src := range matches {
			rel, _ := filepath.Rel(claudeDir, src)
			if !isNeverSync(rel) {
				pairs = append(pairs, FilePair{
					Src:     src,
					Dst:     filepath.Join(repoDir, rel),
					RelPath: rel,
				})
			}
		}
	}

	return filterNeverSync(pairs)
}

// collectPullFiles gathers files from the repo for restoring to ~/.claude/
func collectPullFiles(claudeDir, repoDir string) []FilePair {
	var pairs []FilePair

	// Static files
	for _, rel := range syncFiles {
		src := filepath.Join(repoDir, rel)
		if fileExists(src) {
			pairs = append(pairs, FilePair{
				Src:     src,
				Dst:     filepath.Join(claudeDir, rel),
				RelPath: rel,
			})
		}
	}

	// Glob patterns
	for _, pattern := range syncGlobs {
		matches, _ := filepath.Glob(filepath.Join(repoDir, pattern))
		for _, src := range matches {
			rel, _ := filepath.Rel(repoDir, src)
			if !isNeverSync(rel) {
				pairs = append(pairs, FilePair{
					Src:     src,
					Dst:     filepath.Join(claudeDir, rel),
					RelPath: rel,
				})
			}
		}
	}

	return filterNeverSync(pairs)
}

// collectRepoFiles lists all syncable files currently in the repo (for status)
func collectRepoFiles(repoDir string) map[string]string {
	files := make(map[string]string)

	for _, rel := range syncFiles {
		p := filepath.Join(repoDir, rel)
		if fileExists(p) {
			files[rel] = checksumFile(p)
		}
	}

	for _, pattern := range syncGlobs {
		matches, _ := filepath.Glob(filepath.Join(repoDir, pattern))
		for _, m := range matches {
			rel, _ := filepath.Rel(repoDir, m)
			if !isNeverSync(rel) {
				files[rel] = checksumFile(m)
			}
		}
	}

	return files
}

// collectLocalFiles lists all syncable files in ~/.claude/ (for status)
func collectLocalFiles(claudeDir string) map[string]string {
	files := make(map[string]string)

	for _, rel := range syncFiles {
		p := filepath.Join(claudeDir, rel)
		if fileExists(p) {
			files[rel] = checksumFile(p)
		}
	}

	for _, pattern := range syncGlobs {
		matches, _ := filepath.Glob(filepath.Join(claudeDir, pattern))
		for _, m := range matches {
			rel, _ := filepath.Rel(claudeDir, m)
			if !isNeverSync(rel) {
				files[rel] = checksumFile(m)
			}
		}
	}

	return files
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func checksumFile(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	h := sha256.New()
	io.Copy(h, f)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func isNeverSync(rel string) bool {
	for _, blocked := range neverSync {
		if rel == blocked || strings.HasSuffix(rel, "/"+blocked) {
			return true
		}
	}
	return false
}

func filterNeverSync(pairs []FilePair) []FilePair {
	var filtered []FilePair
	for _, p := range pairs {
		if !isNeverSync(p.RelPath) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}
