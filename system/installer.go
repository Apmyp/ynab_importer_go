package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type Installer struct {
	execPath   string
	workingDir string
	goos       string
	homeDir    string
	apiKey     string
	fileWriter fileWriter
	cmdRunner  commandRunner
}

type fileWriter interface {
	WriteFile(path string, data []byte, perm os.FileMode) error
	Remove(path string) error
	RemoveAll(path string) error
	MkdirAll(path string, perm os.FileMode) error
	CopyFile(src, dst string, perm os.FileMode) error
}

type commandRunner interface {
	Run(name string, args ...string) error
}

type osFileWriter struct{}

func (osFileWriter) WriteFile(path string, data []byte, perm os.FileMode) error {
	return os.WriteFile(path, data, perm)
}

func (osFileWriter) Remove(path string) error {
	return os.Remove(path)
}

func (osFileWriter) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (osFileWriter) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (osFileWriter) CopyFile(src, dst string, perm os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, perm)
}

type execCommandRunner struct{}

func (execCommandRunner) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

func NewInstaller(execPath, workingDir, apiKey string) (*Installer, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	return &Installer{
		execPath:   execPath,
		workingDir: workingDir,
		goos:       runtime.GOOS,
		homeDir:    homeDir,
		apiKey:     apiKey,
		fileWriter: osFileWriter{},
		cmdRunner:  execCommandRunner{},
	}, nil
}

const plistLabel = "com.apmyp.ynab_importer_go"

const shellScriptTemplate = `#!/bin/bash
cd "%s"
exec "$(dirname "$0")/ynab_sync_binary" ynab_sync
`

const appInfoPlistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>ynab_sync</string>
    <key>CFBundleIdentifier</key>
    <string>com.apmyp.ynab_sync</string>
    <key>CFBundleName</key>
    <string>YNAB Sync</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleVersion</key>
    <string>1.0</string>
</dict>
</plist>`

const launchdPlistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>YNAB_API_KEY</key>
        <string>%s</string>
    </dict>
    <key>WorkingDirectory</key>
    <string>%s</string>
    <key>StandardOutPath</key>
    <string>%s/ynab_sync.log</string>
    <key>StandardErrorPath</key>
    <string>%s/ynab_sync_error.log</string>
    <key>StartInterval</key>
    <integer>3600</integer>
    <key>Umask</key>
    <integer>63</integer>
</dict>
</plist>`

func (i *Installer) checkOS() error {
	if i.goos != "darwin" {
		return fmt.Errorf("system installation only supported on macOS, current OS: %s", i.goos)
	}
	return nil
}

func (i *Installer) launchdPlistPath() string {
	return filepath.Join(i.homeDir, "Library/LaunchAgents", plistLabel+".plist")
}

func (i *Installer) appBundlePath() string {
	return filepath.Join(i.workingDir, "ynab_sync.app")
}

func (i *Installer) appContentsPath() string {
	return filepath.Join(i.appBundlePath(), "Contents")
}

func (i *Installer) appMacOSPath() string {
	return filepath.Join(i.appContentsPath(), "MacOS")
}

func (i *Installer) appExecutablePath() string {
	return filepath.Join(i.appMacOSPath(), "ynab_sync")
}

func (i *Installer) appBinaryPath() string {
	return filepath.Join(i.appMacOSPath(), "ynab_sync_binary")
}

func (i *Installer) appInfoPlistPath() string {
	return filepath.Join(i.appContentsPath(), "Info.plist")
}

func (i *Installer) generateScript() string {
	return fmt.Sprintf(shellScriptTemplate, i.workingDir)
}

func (i *Installer) generateAppInfoPlist() string {
	return appInfoPlistTemplate
}

func (i *Installer) generateLaunchdPlist() string {
	return fmt.Sprintf(launchdPlistTemplate,
		plistLabel,
		i.appExecutablePath(),
		i.apiKey,
		i.workingDir,
		i.workingDir,
		i.workingDir,
	)
}

func (i *Installer) checkLaunchd() error {
	_, err := exec.LookPath("launchctl")
	if err != nil {
		return fmt.Errorf("launchctl not found in PATH")
	}
	return nil
}

func (i *Installer) Install() error {
	if err := i.checkOS(); err != nil {
		return err
	}

	if err := i.checkLaunchd(); err != nil {
		return err
	}

	if err := i.fileWriter.MkdirAll(i.appMacOSPath(), 0755); err != nil {
		return fmt.Errorf("failed to create app bundle directories: %w", err)
	}

	appInfoPlistContent := i.generateAppInfoPlist()
	if err := i.fileWriter.WriteFile(i.appInfoPlistPath(), []byte(appInfoPlistContent), 0644); err != nil {
		return fmt.Errorf("failed to write app Info.plist: %w", err)
	}

	if err := i.fileWriter.CopyFile(i.execPath, i.appBinaryPath(), 0755); err != nil {
		return fmt.Errorf("failed to copy binary to app bundle: %w", err)
	}

	executableContent := i.generateScript()
	if err := i.fileWriter.WriteFile(i.appExecutablePath(), []byte(executableContent), 0755); err != nil {
		return fmt.Errorf("failed to write app executable: %w", err)
	}

	launchdPlistPath := i.launchdPlistPath()
	launchdPlistContent := i.generateLaunchdPlist()

	if err := i.fileWriter.WriteFile(launchdPlistPath, []byte(launchdPlistContent), 0644); err != nil {
		return fmt.Errorf("failed to write launchd plist: %w", err)
	}

	if err := i.cmdRunner.Run("launchctl", "load", launchdPlistPath); err != nil {
		return fmt.Errorf("failed to load service: %w", err)
	}

	return nil
}

func (i *Installer) Uninstall() error {
	if err := i.checkOS(); err != nil {
		return err
	}

	launchdPlistPath := i.launchdPlistPath()

	_ = i.cmdRunner.Run("launchctl", "unload", launchdPlistPath)

	if err := i.fileWriter.Remove(launchdPlistPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("service not installed")
		}
		return fmt.Errorf("failed to remove launchd plist: %w", err)
	}

	appBundlePath := i.appBundlePath()
	_ = i.fileWriter.RemoveAll(appBundlePath)

	return nil
}
