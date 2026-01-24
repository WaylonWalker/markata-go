// Package cmd provides the CLI commands for markata-go.
package cmd

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// Update flags.
var (
	updateCheckOnly bool
	updateForce     bool
)

// GitHub API response structures.
type githubRelease struct {
	TagName string        `json:"tag_name"`
	Name    string        `json:"name"`
	Assets  []githubAsset `json:"assets"`
	HTMLURL string        `json:"html_url"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// Constants for update command.
const (
	githubAPIURL = "https://api.github.com/repos/WaylonWalker/markata-go/releases/latest"
	httpTimeout  = 30 * time.Second
	osWindows    = "windows"
	// maxBinarySize is the maximum allowed binary size (100MB) to prevent decompression bombs.
	maxBinarySize = 100 * 1024 * 1024
)

// updateCmd represents the update command.
var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update markata-go to the latest version",
	Long: `Check for and install the latest version of markata-go from GitHub releases.

Examples:
  markata-go update           # Update to latest version
  markata-go update --check   # Check for updates without installing
  markata-go update --force   # Force update even if already on latest`,
	RunE: runUpdate,
}

func init() {
	updateCmd.Flags().BoolVar(&updateCheckOnly, "check", false, "Check for updates without installing")
	updateCmd.Flags().BoolVarP(&updateForce, "force", "f", false, "Force update even if on latest version")
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(_ *cobra.Command, _ []string) error {
	currentVersion := GetVersion()
	fmt.Printf("Current version: %s\n", currentVersion)

	// Fetch latest release info
	release, err := fetchLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to fetch latest release: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	fmt.Printf("Latest version:  %s\n", latestVersion)

	// Compare versions and handle early returns
	if currentVersion == latestVersion && !updateForce {
		fmt.Println("\nYou are already on the latest version!")
		return nil
	}

	if currentVersion != latestVersion {
		fmt.Printf("\nNew version available: %s -> %s\n", currentVersion, latestVersion)
	} else {
		fmt.Println("\nForcing update to same version...")
	}

	if updateCheckOnly {
		fmt.Printf("\nRelease URL: %s\n", release.HTMLURL)
		fmt.Println("Run 'markata-go update' to install.")
		return nil
	}

	// Perform the actual update
	return performUpdate(release, latestVersion)
}

// performUpdate handles downloading, verifying, and installing the update.
func performUpdate(release *githubRelease, latestVersion string) error {
	// Find the appropriate asset for this platform
	assetName, err := getAssetName(latestVersion)
	if err != nil {
		return err
	}

	downloadAsset, checksumAsset, err := findAssets(release, assetName)
	if err != nil {
		return err
	}

	// Download and verify checksum
	fmt.Printf("\nDownloading %s...\n", downloadAsset.Name)
	archivePath, err := downloadFile(downloadAsset.BrowserDownloadURL)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer os.Remove(archivePath)

	if err := verifyChecksum(archivePath, assetName, checksumAsset.BrowserDownloadURL); err != nil {
		return err
	}

	// Extract and install binary
	return installBinary(archivePath, assetName, latestVersion)
}

// findAssets locates the download and checksum assets in the release.
func findAssets(release *githubRelease, assetName string) (download, checksum *githubAsset, err error) {
	for i := range release.Assets {
		if release.Assets[i].Name == assetName {
			download = &release.Assets[i]
		}
		if release.Assets[i].Name == "checksums.txt" {
			checksum = &release.Assets[i]
		}
	}

	if download == nil {
		return nil, nil, fmt.Errorf("could not find release asset for %s/%s (looking for %s)",
			runtime.GOOS, runtime.GOARCH, assetName)
	}

	if checksum == nil {
		return nil, nil, errors.New("could not find checksums.txt in release")
	}

	return download, checksum, nil
}

// verifyChecksum downloads checksums and verifies the archive.
func verifyChecksum(archivePath, assetName, checksumURL string) error {
	fmt.Println("Verifying checksum...")
	checksums, err := downloadChecksums(checksumURL)
	if err != nil {
		return fmt.Errorf("failed to download checksums: %w", err)
	}

	expectedChecksum, ok := checksums[assetName]
	if !ok {
		return fmt.Errorf("checksum not found for %s", assetName)
	}

	actualChecksum, err := calculateSHA256(archivePath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}
	fmt.Println("Checksum verified!")
	return nil
}

// installBinary extracts the binary and replaces the current executable.
func installBinary(archivePath, assetName, latestVersion string) error {
	fmt.Println("Extracting binary...")
	binaryPath, err := extractBinary(archivePath, assetName)
	if err != nil {
		return fmt.Errorf("failed to extract binary: %w", err)
	}
	defer os.Remove(binaryPath)

	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve executable path: %w", err)
	}

	// Replace binary atomically
	fmt.Println("Installing new version...")
	if err := replaceBinary(binaryPath, execPath); err != nil {
		return fmt.Errorf("failed to install: %w", err)
	}

	fmt.Printf("\nSuccessfully updated to version %s!\n", latestVersion)
	return nil
}

func fetchLatestRelease() (*githubRelease, error) {
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubAPIURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", GetShortVersionInfo())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
		}
		return nil, fmt.Errorf("GitHub API returned %d: %s", resp.StatusCode, string(body))
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}

	return &release, nil
}

func getAssetName(version string) (string, error) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Map GOARCH to release naming convention
	arch := goarch
	switch goarch {
	case "amd64":
		arch = "x86_64"
	case "arm":
		arch = "armv7"
	}

	// Build asset name based on OS
	var assetName string
	switch goos {
	case osWindows:
		assetName = fmt.Sprintf("markata-go_%s_%s_%s.zip", version, goos, arch)
	case "linux", "darwin", "freebsd", "android":
		assetName = fmt.Sprintf("markata-go_%s_%s_%s.tar.gz", version, goos, arch)
	default:
		return "", fmt.Errorf("unsupported OS: %s", goos)
	}

	return assetName, nil
}

func downloadFile(url string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", GetShortVersionInfo())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "markata-go-update-*")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

func downloadChecksums(url string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", GetShortVersionInfo())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download checksums: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	checksums := make(map[string]string)
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: "hash  filename" (two spaces)
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			checksums[parts[1]] = parts[0]
		}
	}

	return checksums, nil
}

func calculateSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func extractBinary(archivePath, assetName string) (string, error) {
	if strings.HasSuffix(assetName, ".zip") {
		return extractFromZip(archivePath)
	}
	return extractFromTarGz(archivePath)
}

func extractFromTarGz(archivePath string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	binaryName := "markata-go"
	if runtime.GOOS == osWindows {
		binaryName = "markata-go.exe"
	}

	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}

		if header.Typeflag != tar.TypeReg || filepath.Base(header.Name) != binaryName {
			continue
		}

		tmpFile, err := os.CreateTemp("", "markata-go-binary-*")
		if err != nil {
			return "", err
		}

		// Use LimitReader to prevent decompression bombs
		limited := io.LimitReader(tr, maxBinarySize)
		n, err := io.Copy(tmpFile, limited)
		if err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			return "", err
		}
		tmpFile.Close()

		// Check if we hit the limit (potential bomb)
		if n >= maxBinarySize {
			os.Remove(tmpFile.Name())
			return "", errors.New("binary exceeds maximum allowed size")
		}

		// Make executable
		if err := os.Chmod(tmpFile.Name(), 0o755); err != nil {
			os.Remove(tmpFile.Name())
			return "", err
		}

		return tmpFile.Name(), nil
	}

	return "", fmt.Errorf("binary %s not found in archive", binaryName)
}

func extractFromZip(archivePath string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	binaryName := "markata-go.exe"

	for _, f := range r.File {
		if filepath.Base(f.Name) != binaryName {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return "", err
		}

		tmpFile, err := os.CreateTemp("", "markata-go-binary-*")
		if err != nil {
			rc.Close()
			return "", err
		}

		// Use LimitReader to prevent decompression bombs
		limited := io.LimitReader(rc, maxBinarySize)
		n, err := io.Copy(tmpFile, limited)
		if err != nil {
			tmpFile.Close()
			rc.Close()
			os.Remove(tmpFile.Name())
			return "", err
		}
		tmpFile.Close()
		rc.Close()

		// Check if we hit the limit (potential bomb)
		if n >= maxBinarySize {
			os.Remove(tmpFile.Name())
			return "", errors.New("binary exceeds maximum allowed size")
		}

		return tmpFile.Name(), nil
	}

	return "", fmt.Errorf("binary %s not found in archive", binaryName)
}

func replaceBinary(newBinaryPath, targetPath string) error {
	// Get file info from current executable to preserve permissions
	info, err := os.Stat(targetPath)
	if err != nil {
		return err
	}
	mode := info.Mode()

	// On Windows, we can't replace a running executable directly.
	// We need to rename it first.
	if runtime.GOOS == osWindows {
		return replaceBinaryWindows(newBinaryPath, targetPath)
	}

	// On Unix, use atomic rename
	return replaceBinaryUnix(newBinaryPath, targetPath, mode)
}

func replaceBinaryWindows(newBinaryPath, targetPath string) error {
	oldPath := targetPath + ".old"
	// Remove any existing .old file
	os.Remove(oldPath)

	// Rename current executable
	if err := os.Rename(targetPath, oldPath); err != nil {
		return fmt.Errorf("failed to rename old binary: %w", err)
	}

	// Copy new binary
	if err := copyFile(newBinaryPath, targetPath); err != nil {
		// Try to restore old binary
		if restoreErr := os.Rename(oldPath, targetPath); restoreErr != nil {
			return fmt.Errorf("failed to copy new binary: %w (restore also failed: %w)", err, restoreErr)
		}
		return fmt.Errorf("failed to copy new binary: %w", err)
	}

	// Clean up old binary (may fail if still in use, that's OK)
	os.Remove(oldPath)
	return nil
}

func replaceBinaryUnix(newBinaryPath, targetPath string, mode os.FileMode) error {
	// First copy to a temp file in the same directory (same filesystem)
	dir := filepath.Dir(targetPath)
	tmpFile, err := os.CreateTemp(dir, ".markata-go-update-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	// Copy new binary to temp file
	if err := copyFile(newBinaryPath, tmpPath); err != nil {
		os.Remove(tmpPath)
		return err
	}

	// Set permissions
	if err := os.Chmod(tmpPath, mode); err != nil {
		os.Remove(tmpPath)
		return err
	}

	// Atomic rename
	if err := os.Rename(tmpPath, targetPath); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Close()
}
