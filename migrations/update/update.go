package update

import (
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
	githubAPI = "https://api.github.com/repos/jbarasa/jbmdb/releases/latest"
)

type Release struct {
	TagName    string  `json:"tag_name"`
	Assets     []Asset `json:"assets"`
	Body       string  `json:"body"`
	PreRelease bool    `json:"prerelease"`
}

type Asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
}

// CheckForUpdates checks if there's a new version available
func CheckForUpdates(currentVersion string) (*Release, error) {
	resp, err := http.Get(githubAPI)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %v", err)
	}
	defer resp.Body.Close()

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %v", err)
	}

	if release.TagName == currentVersion {
		return nil, nil // No update available
	}

	return &release, nil
}

// DownloadUpdate downloads and replaces the current binary with the new version
func DownloadUpdate(release *Release) error {
	// Determine which binary to download based on OS
	var binaryName string
	switch runtime.GOOS {
	case "linux":
		binaryName = "jbmdb-linux"
	case "windows":
		binaryName = "jbmdb-windows.exe"
	case "darwin":
		binaryName = "jbmdb-darwin"
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	// Find the correct asset
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == binaryName {
			downloadURL = asset.DownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no binary found for your system")
	}

	// Download the new binary
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %v", err)
	}
	defer resp.Body.Close()

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "jbmdb-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Copy the downloaded binary to temporary file
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return fmt.Errorf("failed to save update: %v", err)
	}
	tmpFile.Close()

	// Make the temporary file executable
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %v", err)
	}

	// Get the path of the current executable
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %v", err)
	}
	currentExe, err = filepath.EvalSymlinks(currentExe)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks: %v", err)
	}

	// Replace the old binary with the new one
	if err := os.Rename(tmpFile.Name(), currentExe); err != nil {
		// If direct rename fails (e.g., on Windows), try copy and remove
		if err := copyFile(tmpFile.Name(), currentExe); err != nil {
			return fmt.Errorf("failed to install update: %v", err)
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// PrintUpdateChangelog prints the changelog/release notes
func PrintUpdateChangelog(release *Release) {
	fmt.Printf("\nChangelog for version %s:\n", release.TagName)
	// Format the body text
	changelog := strings.ReplaceAll(release.Body, "\r\n", "\n")
	fmt.Println(changelog)
}
