package main

import (
	"os"
	"path/filepath"
	"sort"
	"time"
)

const maxBackups = 10

// backupFiles copies files that would be overwritten to a timestamped backup directory.
// Returns the backup directory path and the number of files backed up.
func backupFiles(pairs []FilePair, backupBase string) (string, int, error) {
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	backupDir := filepath.Join(backupBase, timestamp)

	count := 0
	for _, p := range pairs {
		// Only back up if the destination file already exists
		if !fileExists(p.Dst) {
			continue
		}

		backupDst := filepath.Join(backupDir, p.RelPath)
		if err := copyFile(p.Dst, backupDst); err != nil {
			return "", 0, err
		}
		count++
	}

	if count == 0 {
		return "", 0, nil
	}

	return backupDir, count, nil
}

// pruneBackups keeps only the most recent `keep` backup directories.
func pruneBackups(backupBase string, keep int) error {
	if !dirExists(backupBase) {
		return nil
	}

	entries, err := os.ReadDir(backupBase)
	if err != nil {
		return err
	}

	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}

	if len(dirs) <= keep {
		return nil
	}

	sort.Strings(dirs) // timestamps sort lexicographically

	// Remove oldest
	for _, d := range dirs[:len(dirs)-keep] {
		os.RemoveAll(filepath.Join(backupBase, d))
	}

	return nil
}
