package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
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

// parseVersion converts version string like "v1.0.0" to comparable integers
func parseVersion(version string) (major, minor, patch int, err error) {
	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")

	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid version format: %s", version)
	}

	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid major version: %s", parts[0])
	}

	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid minor version: %s", parts[1])
	}

	patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid patch version: %s", parts[2])
	}

	return major, minor, patch, nil
}

// isNewer returns true if version a is newer than version b
func isNewer(a, b string) (bool, error) {
	aMajor, aMinor, aPatch, err := parseVersion(a)
	if err != nil {
		return false, fmt.Errorf("error parsing version %s: %v", a, err)
	}

	bMajor, bMinor, bPatch, err := parseVersion(b)
	if err != nil {
		return false, fmt.Errorf("error parsing version %s: %v", b, err)
	}

	if aMajor > bMajor {
		return true, nil
	}
	if aMajor < bMajor {
		return false, nil
	}

	if aMinor > bMinor {
		return true, nil
	}
	if aMinor < bMinor {
		return false, nil
	}

	return aPatch > bPatch, nil
}

// CheckForUpdates checks if there's a new version available
func CheckForUpdates(currentVersion string) (*Release, error) {
	if currentVersion == "dev" {
		fmt.Println("Development version detected, checking for latest release...")
		currentVersion = "v0.0.0"
	}

	resp, err := http.Get(githubAPI)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %v", err)
	}
	defer resp.Body.Close()

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %v", err)
	}

	// Compare versions
	newer, err := isNewer(release.TagName, currentVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to compare versions: %v", err)
	}

	if !newer {
		fmt.Printf("You are already on the latest version (%s)\n", currentVersion)
		return nil, nil
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

	// Download the new binary with progress
	resp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %v", err)
	}
	defer resp.Body.Close()

	size := resp.ContentLength
	progress := int64(0)
	progressStep := size / 50 // for a 50-character progress bar

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "jbmdb-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Copy the downloaded binary to temporary file with progress
	fmt.Printf("\nDownloading update: [")
	current := int64(0)
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, werr := tmpFile.Write(buf[:n])
			if werr != nil {
				return fmt.Errorf("failed to write update: %v", werr)
			}
			current += int64(n)
			for current >= progress+progressStep && progress < size {
				fmt.Print("=")
				progress += progressStep
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to download update: %v", err)
		}
	}
	fmt.Println("]")
	tmpFile.Close()

	// Make the temporary file executable
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %v", err)
	}

	// Get the path to the current executable
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %v", err)
	}

	// Try to replace the binary
	err = os.Rename(tmpFile.Name(), execPath)
	if err != nil {
		if strings.Contains(err.Error(), "text file busy") {
			// If binary is busy, try to restart automatically
			fmt.Println("\nCurrent binary is running. Attempting automatic restart...")

			// Get the absolute path to the new binary
			newBinaryPath, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get new binary path: %v", err)
			}
			newBinaryPath = newBinaryPath + "/" + tmpFile.Name()

			// Create the restart script
			scriptContent := fmt.Sprintf(`#!/bin/bash
sleep 1
mv "%s" "%s"
chmod +x "%s"
"%s" "$@"
`, newBinaryPath, execPath, execPath, execPath)

			restartScript, err := os.CreateTemp("", "jbmdb-restart-*.sh")
			if err != nil {
				return fmt.Errorf("failed to create restart script: %v", err)
			}
			defer os.Remove(restartScript.Name())

			if err := os.WriteFile(restartScript.Name(), []byte(scriptContent), 0755); err != nil {
				return fmt.Errorf("failed to write restart script: %v", err)
			}

			// Execute the restart script and exit current process
			if err := syscall.Exec(restartScript.Name(), []string{restartScript.Name()}, os.Environ()); err != nil {
				return fmt.Errorf("failed to execute restart script: %v", err)
			}

			// We won't reach here if exec succeeds
			return nil
		}
		return fmt.Errorf("failed to install update: %v", err)
	}

	fmt.Println("Update successfully installed!")
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
