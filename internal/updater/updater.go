package updater

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"proxy-core/internal/sysinfo"
)

const RepoOwner = "niuniu06"
const RepoName = "MinerLink-Proxy"

type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

type UpdateStatus struct {
	HasUpdate     bool   `json:"hasUpdate"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	Changelog     string `json:"changelog"`
}

// CheckForUpdates fetches the latest release from GitHub API
func CheckForUpdates() (*UpdateStatus, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", RepoOwner, RepoName)
	client := &http.Client{Timeout: 8 * time.Second}
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	// Add User-Agent header (GitHub API requires it)
	req.Header.Set("User-Agent", "go-proxy-updater")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	current := sysinfo.ProxyVersion
	latest := release.TagName

	// Simple check: if they don't match, propose update
	hasUpdate := false
	if latest != "" && normalizeVersion(latest) != normalizeVersion(current) {
		hasUpdate = true
	}

	return &UpdateStatus{
		HasUpdate:      hasUpdate,
		CurrentVersion: current,
		LatestVersion:  latest,
		Changelog:      release.Body,
	}, nil
}

// normalizeVersion strips 'v' and '-beta' for comparison
func normalizeVersion(v string) string {
	v = strings.ToLower(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.Split(v, "-")[0]
	return strings.TrimSpace(v)
}

// copyFile is a helper to copy a file when os.Rename fails across devices
func copyFile(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

// StartUpgrade handles downloading the matched binary and executing hot reload
func StartUpgrade() error {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", RepoOwner, RepoName)
	client := &http.Client{Timeout: 8 * time.Second}
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "go-proxy-updater")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch release assets: status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return err
	}

	// Find the matching asset for the current OS
	downloadURL := ""
	assetName := ""
	isWindows := runtime.GOOS == "windows"

	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		if isWindows {
			if strings.HasSuffix(name, ".exe") && !strings.Contains(name, "tunnel") {
				downloadURL = asset.BrowserDownloadURL
				assetName = asset.Name
				break
			}
		} else {
			// Linux: look for linux target but exclude tunnel
			if (strings.Contains(name, "linux") || !strings.Contains(name, ".")) && !strings.Contains(name, "tunnel") {
				downloadURL = asset.BrowserDownloadURL
				assetName = asset.Name
				break
			}
		}
	}

	if downloadURL == "" {
		return errors.New("no matching binary found for current operating system in release assets")
	}

	// Download new binary to a temp file
	tempFile := filepath.Join(os.TempDir(), assetName)
	if err := downloadFile(tempFile, downloadURL); err != nil {
		return fmt.Errorf("failed to download update: %v", err)
	}

	// Extract the binary from the zip file
	extractedBinPath := filepath.Join(os.TempDir(), "MinerLink-Proxy-update-bin")
	if isWindows {
		extractedBinPath += ".exe"
	}
	if err := extractBinaryFromZip(tempFile, extractedBinPath, isWindows); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("failed to extract update binary from zip: %v", err)
	}
	_ = os.Remove(tempFile) // delete zip after extraction

	// Trigger the platform specific replacement in a separate goroutine after returning HTTP 200
	go func() {
		time.Sleep(1 * time.Second) // Let API response finish
		if isWindows {
			_ = upgradeWindows(extractedBinPath)
		} else {
			_ = upgradeLinux(extractedBinPath)
		}
	}()

	return nil
}

func extractBinaryFromZip(zipPath string, destPath string, isWindows bool) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		name := strings.ToLower(f.Name)
		// Match the proxy executable
		if isWindows && strings.HasSuffix(name, ".exe") && !strings.Contains(name, "tunnel") {
			return extractSingleFile(f, destPath)
		} else if !isWindows && strings.Contains(name, "linux") && !strings.Contains(name, "tunnel") {
			return extractSingleFile(f, destPath)
		}
	}
	return errors.New("could not find proxy executable in zip file")
}

func extractSingleFile(f *zip.File, destPath string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}

func downloadFile(filepath string, url string) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func upgradeLinux(newBinaryPath string) error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	oldPath := execPath + ".old"
	_ = os.Remove(oldPath) // Remove old backup if it exists

	// Rename current binary to backup name
	if err := os.Rename(execPath, oldPath); err != nil {
		return fmt.Errorf("failed to backup current binary: %v", err)
	}

	// Rename new binary to target path
	if err := os.Rename(newBinaryPath, execPath); err != nil {
		// If rename fails (e.g. cross-device link), fallback to copy and delete
		if copyErr := copyFile(newBinaryPath, execPath); copyErr != nil {
			// Restore backup on failure
			_ = os.Rename(oldPath, execPath)
			return fmt.Errorf("failed to place new binary (rename: %v, copy: %v)", err, copyErr)
		}
		// Clean up the temp binary if copy succeeded
		_ = os.Remove(newBinaryPath)
	}

	// Set execution permissions
	if err := os.Chmod(execPath, 0755); err != nil {
		return fmt.Errorf("failed to set execution permissions: %v", err)
	}

	// Seamlessly replace the process image using syscall.Exec
	// This inherits all sockets (if keep-alive or running sockets aren't closed),
	// but simple restart is also fine as systemd will auto restart if it exits.
	// syscall.Exec is cleaner for a true in-place hot reload.
	args := os.Args
	env := os.Environ()
	
	err = syscall.Exec(execPath, args, env)
	if err != nil {
		// If exec fails, exit and let supervisord/systemd handle it
		os.Exit(0)
	}
	return nil
}

func upgradeWindows(newBinaryPath string) error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}

	// On Windows, running executable is locked. We drop a .bat script to replace and restart.
	batContent := fmt.Sprintf(`@echo off
title Go-Proxy Auto Updater
echo Waiting for proxy process to release file lock...
timeout /t 2 /nobreak > nul
copy /y "%s" "%s"
if errorlevel 1 (
    echo Error replacing proxy.exe, retrying in 3 seconds...
    timeout /t 3 /nobreak > nul
    copy /y "%s" "%s"
)
del "%s"
echo Starting updated proxy...
start "" "%s"
(goto) 2>nul & del "%s"
`, newBinaryPath, execPath, newBinaryPath, execPath, newBinaryPath, execPath, filepath.Join(filepath.Dir(execPath), "upgrade.bat"))

	batPath := filepath.Join(filepath.Dir(execPath), "upgrade.bat")
	err = os.WriteFile(batPath, []byte(batContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write update script: %v", err)
	}

	// Spawn the batch file silently using cmd
	cmd := exec.Command("cmd.exe", "/c", "start", "upgrade.bat")
	cmd.Dir = filepath.Dir(execPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to run update script: %v", err)
	}

	// Terminate parent process so batch file can write over it
	os.Exit(0)
	return nil
}
