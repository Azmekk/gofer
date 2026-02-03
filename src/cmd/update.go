package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	repoOwner = "Azmekk"
	repoName  = "gofer"
	apiURL    = "https://api.github.com/repos/" + repoOwner + "/" + repoName + "/releases/latest"
)

type githubRelease struct {
	TagName string `json:"tag_name"`
}

func selfUpdate() error {
	fmt.Println("Checking for updates...")

	release, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if release.TagName == Version || release.TagName == "v"+Version {
		fmt.Printf("Already up to date (%s).\n", Version)
		return nil
	}

	fmt.Printf("New version available: %s (current: %s)\n", release.TagName, Version)

	goos := runtime.GOOS
	goarch := runtime.GOARCH
	suffix := ""
	if goos == "windows" {
		suffix = ".exe"
	}

	binaryName := fmt.Sprintf("gofer-%s-%s%s", goos, goarch, suffix)
	downloadURL := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s",
		repoOwner, repoName, release.TagName, binaryName)
	checksumsURL := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/checksums.txt",
		repoOwner, repoName, release.TagName)

	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to determine executable path: %w", err)
	}
	currentExe, err = filepath.EvalSymlinks(currentExe)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	fmt.Printf("Downloading %s...\n", binaryName)

	tempFile := currentExe + ".new"
	if err := downloadFile(tempFile, downloadURL); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to download update: %w", err)
	}

	// Verify checksum
	if err := verifyChecksum(tempFile, binaryName, checksumsURL); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("checksum verification failed: %w", err)
	}

	// Replace the current executable
	if err := replaceBinary(currentExe, tempFile); err != nil {
		os.Remove(tempFile)
		return fmt.Errorf("failed to replace executable: %w", err)
	}

	fmt.Printf("Successfully updated to %s.\n", release.TagName)
	return nil
}

func fetchLatestRelease() (*githubRelease, error) {
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "gofer-updater")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

func downloadFile(dest, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func verifyChecksum(filePath, binaryName, checksumsURL string) error {
	resp, err := http.Get(checksumsURL)
	if err != nil {
		fmt.Println("Warning: could not download checksums, skipping verification.")
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Warning: could not download checksums, skipping verification.")
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Warning: could not read checksums, skipping verification.")
		return nil
	}

	var expectedHash string
	for _, line := range strings.Split(string(body), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == binaryName {
			expectedHash = parts[0]
			break
		}
	}

	if expectedHash == "" {
		return fmt.Errorf("binary %s not found in checksums.txt", binaryName)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	actualHash := hex.EncodeToString(h.Sum(nil))
	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	fmt.Println("Checksum verified.")
	return nil
}

func replaceBinary(currentExe, tempFile string) error {
	if runtime.GOOS == "windows" {
		oldPath := currentExe + ".old"
		os.Remove(oldPath)
		if err := os.Rename(currentExe, oldPath); err != nil {
			return err
		}
		if err := os.Rename(tempFile, currentExe); err != nil {
			// Try to restore the old binary
			os.Rename(oldPath, currentExe)
			return err
		}
		return nil
	}

	return os.Rename(tempFile, currentExe)
}
