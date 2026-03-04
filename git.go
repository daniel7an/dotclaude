package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func gitClone(url, dest string) error {
	cmd := exec.Command("git", "clone", url, dest)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %s\n%s", err, out)
	}
	return nil
}

func gitAdd(repoPath string) error {
	cmd := exec.Command("git", "-C", repoPath, "add", "-A")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git add failed: %s\n%s", err, out)
	}
	return nil
}

func gitCommit(repoPath, msg string) error {
	cmd := exec.Command("git", "-C", repoPath, "commit", "-m", msg)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// "nothing to commit" is not an error for us
		if strings.Contains(string(out), "nothing to commit") {
			return nil
		}
		return fmt.Errorf("git commit failed: %s\n%s", err, out)
	}
	return nil
}

func gitPush(repoPath string) error {
	cmd := exec.Command("git", "-C", repoPath, "push")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push failed: %s\n%s", err, out)
	}
	return nil
}

func gitPull(repoPath string) error {
	cmd := exec.Command("git", "-C", repoPath, "pull")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull failed: %s\n%s", err, out)
	}
	return nil
}

func gitHasChanges(repoPath string) bool {
	cmd := exec.Command("git", "-C", repoPath, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) != ""
}

func gitStatusOutput(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git status failed: %s", err)
	}
	return string(out), nil
}
