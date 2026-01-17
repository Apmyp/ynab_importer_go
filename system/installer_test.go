package system

import (
	"fmt"
	"os"
	"testing"
)

// Mock implementations for testing
type mockFileWriter struct {
	writeErr     error
	removeErr    error
	writtenFiles map[string][]byte
	removedFiles []string
	createdDirs  []string
}

func (m *mockFileWriter) WriteFile(path string, data []byte, perm os.FileMode) error {
	if m.writeErr != nil {
		return m.writeErr
	}
	if m.writtenFiles == nil {
		m.writtenFiles = make(map[string][]byte)
	}
	m.writtenFiles[path] = data
	return nil
}

func (m *mockFileWriter) Remove(path string) error {
	if m.removeErr != nil {
		return m.removeErr
	}
	m.removedFiles = append(m.removedFiles, path)
	return nil
}

func (m *mockFileWriter) RemoveAll(path string) error {
	return m.Remove(path)
}

func (m *mockFileWriter) MkdirAll(path string, perm os.FileMode) error {
	if m.createdDirs == nil {
		m.createdDirs = make([]string, 0)
	}
	m.createdDirs = append(m.createdDirs, path)
	return nil
}

func (m *mockFileWriter) CopyFile(src, dst string, perm os.FileMode) error {
	if m.writtenFiles == nil {
		m.writtenFiles = make(map[string][]byte)
	}
	m.writtenFiles[dst] = []byte("binary-content")
	return nil
}

type mockCommandRunner struct {
	runErr   error
	commands [][]string
}

func (m *mockCommandRunner) Run(name string, args ...string) error {
	if m.runErr != nil {
		return m.runErr
	}
	cmd := append([]string{name}, args...)
	m.commands = append(m.commands, cmd)
	return nil
}

func TestCheckOS_Darwin(t *testing.T) {
	installer := &Installer{
		goos: "darwin",
	}

	err := installer.checkOS()
	if err != nil {
		t.Errorf("checkOS() on darwin should not return error, got: %v", err)
	}
}

func TestCheckOS_NonDarwin(t *testing.T) {
	tests := []struct {
		name string
		goos string
	}{
		{"linux", "linux"},
		{"windows", "windows"},
		{"freebsd", "freebsd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installer := &Installer{
				goos: tt.goos,
			}

			err := installer.checkOS()
			if err == nil {
				t.Errorf("checkOS() on %s should return error", tt.goos)
			}
		})
	}
}

func TestGenerateLaunchdPlist(t *testing.T) {
	installer := &Installer{
		execPath:   "/usr/local/bin/ynab_importer_go",
		workingDir: "/Users/test/config",
		apiKey:     "test-api-key-12345",
	}

	plist := installer.generateLaunchdPlist()

	// Check for required elements
	requiredElements := []string{
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?>",
		"<plist version=\"1.0\">",
		"<key>Label</key>",
		"<string>com.apmyp.ynab_importer_go</string>",
		"<key>ProgramArguments</key>",
		"<string>/Users/test/config/ynab_sync.app/Contents/MacOS/ynab_sync</string>",
		"<key>EnvironmentVariables</key>",
		"<key>YNAB_API_KEY</key>",
		"<string>test-api-key-12345</string>",
		"<key>WorkingDirectory</key>",
		"<string>/Users/test/config</string>",
		"<key>StandardOutPath</key>",
		"<string>/Users/test/config/ynab_sync.log</string>",
		"<key>StandardErrorPath</key>",
		"<string>/Users/test/config/ynab_sync_error.log</string>",
		"<key>StartInterval</key>",
		"<integer>3600</integer>",
	}

	for _, elem := range requiredElements {
		if !contains(plist, elem) {
			t.Errorf("generateLaunchdPlist() missing required element: %s", elem)
		}
	}
}

func TestGenerateAppInfoPlist(t *testing.T) {
	installer := &Installer{}

	plist := installer.generateAppInfoPlist()

	requiredElements := []string{
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?>",
		"<key>CFBundleExecutable</key>",
		"<string>ynab_sync</string>",
		"<key>CFBundleIdentifier</key>",
		"<string>com.apmyp.ynab_sync</string>",
		"<key>CFBundleName</key>",
		"<string>YNAB Sync</string>",
		"<key>CFBundlePackageType</key>",
		"<string>APPL</string>",
	}

	for _, elem := range requiredElements {
		if !contains(plist, elem) {
			t.Errorf("generateAppInfoPlist() missing required element: %s", elem)
		}
	}
}

func TestAppBundlePath(t *testing.T) {
	installer := &Installer{
		workingDir: "/Users/test/config",
	}

	path := installer.appBundlePath()
	expected := "/Users/test/config/ynab_sync.app"

	if path != expected {
		t.Errorf("appBundlePath() = %s, want %s", path, expected)
	}
}

func TestAppExecutablePath(t *testing.T) {
	installer := &Installer{
		workingDir: "/Users/test/config",
	}

	path := installer.appExecutablePath()
	expected := "/Users/test/config/ynab_sync.app/Contents/MacOS/ynab_sync"

	if path != expected {
		t.Errorf("appExecutablePath() = %s, want %s", path, expected)
	}
}

func TestGenerateScript(t *testing.T) {
	installer := &Installer{
		workingDir: "/Users/test/config",
	}

	script := installer.generateScript()

	requiredElements := []string{
		"#!/bin/bash",
		"cd \"/Users/test/config\"",
		"exec \"$(dirname \"$0\")/ynab_sync_binary\" ynab_sync",
	}

	for _, elem := range requiredElements {
		if !contains(script, elem) {
			t.Errorf("generateScript() missing required element: %s", elem)
		}
	}

	if contains(script, "YNAB_API_KEY") {
		t.Error("generateScript() should not contain YNAB_API_KEY")
	}
}

func TestLaunchdPlistPath(t *testing.T) {
	installer := &Installer{
		homeDir: "/Users/testuser",
	}

	path := installer.launchdPlistPath()
	expected := "/Users/testuser/Library/LaunchAgents/com.apmyp.ynab_importer_go.plist"

	if path != expected {
		t.Errorf("launchdPlistPath() = %s, want %s", path, expected)
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestInstall_Success(t *testing.T) {
	mockWriter := &mockFileWriter{}
	mockRunner := &mockCommandRunner{}

	installer := &Installer{
		execPath:   "/usr/local/bin/ynab_importer_go",
		workingDir: "/Users/test/config",
		apiKey:     "test-api-key",
		goos:       "darwin",
		homeDir:    "/Users/test",
		fileWriter: mockWriter,
		cmdRunner:  mockRunner,
	}

	err := installer.Install()
	if err != nil {
		t.Errorf("Install() should not return error, got: %v", err)
	}

	// Check app bundle directory was created
	expectedDir := "/Users/test/config/ynab_sync.app/Contents/MacOS"
	if len(mockWriter.createdDirs) != 1 || mockWriter.createdDirs[0] != expectedDir {
		t.Errorf("Install() should create directory %s, got %v", expectedDir, mockWriter.createdDirs)
	}

	// Check app Info.plist was written
	appInfoPlistPath := "/Users/test/config/ynab_sync.app/Contents/Info.plist"
	if _, ok := mockWriter.writtenFiles[appInfoPlistPath]; !ok {
		t.Errorf("Install() did not write app Info.plist to %s", appInfoPlistPath)
	}

	// Check binary was copied into app bundle
	binaryPath := "/Users/test/config/ynab_sync.app/Contents/MacOS/ynab_sync_binary"
	if _, ok := mockWriter.writtenFiles[binaryPath]; !ok {
		t.Errorf("Install() did not copy binary to %s", binaryPath)
	}

	// Check app executable was written
	executablePath := "/Users/test/config/ynab_sync.app/Contents/MacOS/ynab_sync"
	if _, ok := mockWriter.writtenFiles[executablePath]; !ok {
		t.Errorf("Install() did not write app executable to %s", executablePath)
	}

	// Check launchd plist was written
	launchdPlistPath := "/Users/test/Library/LaunchAgents/com.apmyp.ynab_importer_go.plist"
	if _, ok := mockWriter.writtenFiles[launchdPlistPath]; !ok {
		t.Errorf("Install() did not write launchd plist to %s", launchdPlistPath)
	}

	// Check launchctl load was called
	if len(mockRunner.commands) != 1 {
		t.Errorf("Install() should call launchctl load once, got %d calls", len(mockRunner.commands))
	} else {
		cmd := mockRunner.commands[0]
		if len(cmd) != 3 || cmd[0] != "launchctl" || cmd[1] != "load" || cmd[2] != launchdPlistPath {
			t.Errorf("Install() should call 'launchctl load %s', got %v", launchdPlistPath, cmd)
		}
	}
}

func TestInstall_NonDarwin(t *testing.T) {
	installer := &Installer{
		goos: "linux",
	}

	err := installer.Install()
	if err == nil {
		t.Error("Install() on non-darwin should return error")
	}
}

func TestInstall_FileWriteError(t *testing.T) {
	mockWriter := &mockFileWriter{
		writeErr: fmt.Errorf("permission denied"),
	}
	mockRunner := &mockCommandRunner{}

	installer := &Installer{
		execPath:   "/usr/local/bin/ynab_importer_go",
		workingDir: "/Users/test/config",
		apiKey:     "test-api-key",
		goos:       "darwin",
		homeDir:    "/Users/test",
		fileWriter: mockWriter,
		cmdRunner:  mockRunner,
	}

	err := installer.Install()
	if err == nil {
		t.Error("Install() should return error when file write fails")
	}
}

func TestInstall_LaunchctlError(t *testing.T) {
	mockWriter := &mockFileWriter{}
	mockRunner := &mockCommandRunner{
		runErr: fmt.Errorf("launchctl failed"),
	}

	installer := &Installer{
		execPath:   "/usr/local/bin/ynab_importer_go",
		workingDir: "/Users/test/config",
		apiKey:     "test-api-key",
		goos:       "darwin",
		homeDir:    "/Users/test",
		fileWriter: mockWriter,
		cmdRunner:  mockRunner,
	}

	err := installer.Install()
	if err == nil {
		t.Error("Install() should return error when launchctl fails")
	}
}

func TestUninstall_Success(t *testing.T) {
	mockWriter := &mockFileWriter{}
	mockRunner := &mockCommandRunner{}

	installer := &Installer{
		goos:       "darwin",
		homeDir:    "/Users/test",
		workingDir: "/Users/test/config",
		fileWriter: mockWriter,
		cmdRunner:  mockRunner,
	}

	err := installer.Uninstall()
	if err != nil {
		t.Errorf("Uninstall() should not return error, got: %v", err)
	}

	// Check launchctl unload was called
	if len(mockRunner.commands) != 1 {
		t.Errorf("Uninstall() should call launchctl unload once, got %d calls", len(mockRunner.commands))
	}

	// Check files were removed (launchd plist and app bundle)
	launchdPlistPath := "/Users/test/Library/LaunchAgents/com.apmyp.ynab_importer_go.plist"
	appBundlePath := "/Users/test/config/ynab_sync.app"
	if len(mockWriter.removedFiles) != 2 {
		t.Errorf("Uninstall() should remove 2 items, got %d: %v", len(mockWriter.removedFiles), mockWriter.removedFiles)
	}
	if mockWriter.removedFiles[0] != launchdPlistPath {
		t.Errorf("Uninstall() should remove launchd plist at %s first, got %s", launchdPlistPath, mockWriter.removedFiles[0])
	}
	if mockWriter.removedFiles[1] != appBundlePath {
		t.Errorf("Uninstall() should remove app bundle at %s second, got %s", appBundlePath, mockWriter.removedFiles[1])
	}
}

func TestUninstall_NonDarwin(t *testing.T) {
	installer := &Installer{
		goos: "windows",
	}

	err := installer.Uninstall()
	if err == nil {
		t.Error("Uninstall() on non-darwin should return error")
	}
}

func TestUninstall_NotInstalled(t *testing.T) {
	mockWriter := &mockFileWriter{
		removeErr: os.ErrNotExist,
	}
	mockRunner := &mockCommandRunner{}

	installer := &Installer{
		goos:       "darwin",
		homeDir:    "/Users/test",
		fileWriter: mockWriter,
		cmdRunner:  mockRunner,
	}

	err := installer.Uninstall()
	if err == nil {
		t.Error("Uninstall() should return error when service not installed")
	}
}

func TestNewInstaller(t *testing.T) {
	installer, err := NewInstaller("/usr/bin/test", "/test/working/dir", "test-api-key")
	if err != nil {
		t.Errorf("NewInstaller() should not return error, got: %v", err)
	}
	if installer == nil {
		t.Error("NewInstaller() should return non-nil installer")
	}
	if installer.execPath != "/usr/bin/test" {
		t.Errorf("NewInstaller() execPath = %s, want /usr/bin/test", installer.execPath)
	}
	if installer.workingDir != "/test/working/dir" {
		t.Errorf("NewInstaller() workingDir = %s, want /test/working/dir", installer.workingDir)
	}
	if installer.apiKey != "test-api-key" {
		t.Errorf("NewInstaller() apiKey = %s, want test-api-key", installer.apiKey)
	}
}
