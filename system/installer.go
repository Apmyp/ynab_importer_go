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
	fileWriter fileWriter
	cmdRunner  commandRunner
}

type fileWriter interface {
	WriteFile(path string, data []byte, perm os.FileMode) error
	Remove(path string) error
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

type execCommandRunner struct{}

func (execCommandRunner) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

func NewInstaller(execPath, workingDir string) (*Installer, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	return &Installer{
		execPath:   execPath,
		workingDir: workingDir,
		goos:       runtime.GOOS,
		homeDir:    homeDir,
		fileWriter: osFileWriter{},
		cmdRunner:  execCommandRunner{},
	}, nil
}

const plistLabel = "com.apmyp.ynab_importer_go"
const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
        <string>%s</string>
        <string>ynab_sync</string>
    </array>
    <key>WorkingDirectory</key>
    <string>%s</string>
    <key>StandardOutPath</key>
    <string>%s/ynab_sync.log</string>
    <key>StandardErrorPath</key>
    <string>%s/ynab_sync_error.log</string>
    <key>StartInterval</key>
    <integer>3600</integer>
</dict>
</plist>`

func (i *Installer) checkOS() error {
	if i.goos != "darwin" {
		return fmt.Errorf("system installation only supported on macOS, current OS: %s", i.goos)
	}
	return nil
}

func (i *Installer) plistPath() string {
	return filepath.Join(i.homeDir, "Library/LaunchAgents", plistLabel+".plist")
}

func (i *Installer) generatePlist() string {
	return fmt.Sprintf(plistTemplate,
		plistLabel,
		i.execPath,
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

	plistPath := i.plistPath()
	plistContent := i.generatePlist()

	if err := i.fileWriter.WriteFile(plistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("failed to write plist: %w", err)
	}

	if err := i.cmdRunner.Run("launchctl", "load", plistPath); err != nil {
		return fmt.Errorf("failed to load service: %w", err)
	}

	return nil
}

func (i *Installer) Uninstall() error {
	if err := i.checkOS(); err != nil {
		return err
	}

	plistPath := i.plistPath()

	_ = i.cmdRunner.Run("launchctl", "unload", plistPath)

	if err := i.fileWriter.Remove(plistPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("service not installed")
		}
		return fmt.Errorf("failed to remove plist: %w", err)
	}

	return nil
}
