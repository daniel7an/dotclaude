package main

import (
	"os"
	"path/filepath"
	"strings"
)

// ProjectMapping links a repo alias to a local encoded project directory
type ProjectMapping struct {
	Alias      string // e.g., "yerevan-eye"
	LocalPath  string // e.g., "/home/daniielyan/projects/yerevan-eye"
	EncodedDir string // e.g., "-home-daniielyan-projects-yerevan-eye"
}

// discoverProjects scans ~/.claude/projects/*/memory/MEMORY.md and builds mappings
func discoverProjects(claudeDir string) []ProjectMapping {
	pattern := filepath.Join(claudeDir, "projects", "*", "memory", "MEMORY.md")
	matches, _ := filepath.Glob(pattern)

	var mappings []ProjectMapping
	for _, m := range matches {
		// Extract the encoded dir name
		// m = <claudeDir>/projects/<encodedDir>/memory/MEMORY.md
		rel, _ := filepath.Rel(filepath.Join(claudeDir, "projects"), m)
		parts := strings.SplitN(rel, string(filepath.Separator), 2)
		encodedDir := parts[0]

		alias := aliasFromEncoded(encodedDir)
		localPath := decodeToPath(encodedDir)

		mappings = append(mappings, ProjectMapping{
			Alias:      alias,
			LocalPath:  localPath,
			EncodedDir: encodedDir,
		})
	}

	return mappings
}

// aliasFromEncoded derives a short alias from an encoded directory name.
// It tries to find the real path on disk and uses the basename.
// Falls back to the last meaningful segment of the encoded name.
func aliasFromEncoded(encoded string) string {
	realPath := decodeToPath(encoded)

	// If the real path exists, use its basename
	if dirExists(realPath) {
		return filepath.Base(realPath)
	}

	// Fall back: extract last segment from encoded name
	// encoded = "-home-daniielyan-projects-yerevan-eye"
	// Split by "-" and try to find meaningful suffix
	return lastMeaningfulSegment(encoded)
}

// decodeToPath converts an encoded dir name back to an absolute path.
// "-home-daniielyan-projects-foo" → "/home/daniielyan/projects/foo"
func decodeToPath(encoded string) string {
	// Remove leading "-" and replace remaining "-" with "/"
	// But we need to be careful: directory names can contain hyphens.
	// The encoding is: replace "/" with "-" and drop leading "-".
	// So "-home-daniielyan-projects-yerevan--eye" isn't a thing —
	// Claude uses simple replacement: /home/dan/foo → -home-dan-foo
	//
	// To decode, we try progressively: start from /home/<user>/... and
	// find the longest existing path prefix.
	if !strings.HasPrefix(encoded, "-") {
		encoded = "-" + encoded
	}

	// Simple approach: replace all "-" with "/", giving us a candidate path
	candidate := strings.ReplaceAll(encoded, "-", "/")
	if dirExists(candidate) {
		return candidate
	}

	// Smart approach: try to reconstruct by finding existing directories
	// Split into segments and try greedy matching
	segments := strings.Split(strings.TrimPrefix(encoded, "-"), "-")
	return greedyPathResolve(segments)
}

// greedyPathResolve tries to reconstruct a real path from segments,
// handling directory names that contain hyphens.
func greedyPathResolve(segments []string) string {
	if len(segments) == 0 {
		return "/"
	}

	// Start from root, greedily try to build the longest matching path
	return greedyResolveFrom("/", segments)
}

func greedyResolveFrom(base string, segments []string) string {
	if len(segments) == 0 {
		return base
	}

	// Try joining progressively more segments as a single directory name
	for i := len(segments); i >= 1; i-- {
		candidate := strings.Join(segments[:i], "-")
		fullPath := filepath.Join(base, candidate)

		if dirExists(fullPath) {
			if i == len(segments) {
				return fullPath
			}
			// Recurse with remaining segments
			result := greedyResolveFrom(fullPath, segments[i:])
			if dirExists(result) {
				return result
			}
		}
	}

	// Nothing matched on disk; reconstruct best-effort
	// Just join everything with "/" as the naive decode
	parts := make([]string, len(segments))
	copy(parts, segments)
	return filepath.Join(base, filepath.Join(parts...))
}

// lastMeaningfulSegment extracts the last meaningful path segment from an encoded name.
// Skips common prefixes like "home", username, "projects", "code", etc.
func lastMeaningfulSegment(encoded string) string {
	segments := strings.Split(strings.TrimPrefix(encoded, "-"), "-")

	// Common prefixes to skip
	skip := map[string]bool{
		"home": true, "root": true, "usr": true,
		"projects": true, "code": true, "src": true,
		"repos": true, "workspace": true, "work": true,
		"go": true, "github.com": true, "gitlab.com": true,
	}

	// Also skip what looks like a username (second segment after "home")
	if len(segments) >= 2 && segments[0] == "home" {
		skip[segments[1]] = true
	}

	// Find the last non-skipped segment
	for i := len(segments) - 1; i >= 0; i-- {
		if !skip[segments[i]] && segments[i] != "" {
			return segments[i]
		}
	}

	// Fallback: use the last segment regardless
	if len(segments) > 0 {
		return segments[len(segments)-1]
	}
	return encoded
}

// encodePath converts an absolute path to Claude's encoded directory format.
// /home/dan/projects/foo → -home-dan-projects-foo
func encodePath(absPath string) string {
	return strings.ReplaceAll(absPath, "/", "-")
}

// findEncodedDirForAlias searches ~/.claude/projects/ for an encoded dir
// that matches the given alias.
func findEncodedDirForAlias(claudeDir, alias string) string {
	projectsDir := filepath.Join(claudeDir, "projects")
	if !dirExists(projectsDir) {
		return ""
	}

	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return ""
	}

	// Load config overrides
	cfg := loadConfig()
	if override, ok := cfg.Projects[alias]; ok {
		encoded := encodePath(override)
		// Check if this encoded dir exists
		for _, e := range entries {
			if e.IsDir() && e.Name() == encoded {
				return encoded
			}
		}
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// Check if this encoded dir's alias matches
		if aliasFromEncoded(e.Name()) == alias {
			return e.Name()
		}
	}

	return ""
}
