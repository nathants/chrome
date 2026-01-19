// launch provides Chrome launcher command.
package launch

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/nathants/chrome/lib"
)

const defaultDebugPort = 9222

func init() {
	lib.Commands["launch"] = launchChrome
	lib.Args["launch"] = launchArgs{}
}

type launchArgs struct {
	LogFile     string `arg:"--log-file" help:"Log file for Chrome output (default: <tmp>/chrome-launch.log)"`
	UserDataDir string `arg:"--user-data-dir" help:"Chrome user data dir (default: ~/.chrome on Unix, C:\\temp\\chrome on WSL)"`
}

func (launchArgs) Description() string {
	return `launch - Launch Chrome with remote debugging

Launches Chrome with remote debugging enabled on port 9222.

Security: remote debugging grants full control of Chrome. This command binds the
debugging endpoint to localhost only.

Example:
  chrome launch
  chrome launch --log-file /tmp/chrome.log
  chrome launch --user-data-dir ~/.chrome`
}

func launchChrome() {
	var args launchArgs
	arg.MustParse(&args)

	if lib.IsChromeRunning() {
		fmt.Fprintf(os.Stderr, "Chrome already running on port %d. Use:\n", defaultDebugPort)
		fmt.Fprintf(os.Stderr, "  chrome list                    # See open tabs\n")
		fmt.Fprintf(os.Stderr, "  chrome newtab <url>            # Open new tab\n")
		fmt.Fprintf(os.Stderr, "  chrome -t <url-prefix> <cmd>   # Target existing tab\n")
		os.Exit(0)
	}

	chromePath := findChrome()
	if chromePath == "" {
		fmt.Fprintln(os.Stderr, "error: Chrome not found. Install Google Chrome or Chromium.")
		os.Exit(1)
	}

	userDataDir := strings.TrimSpace(args.UserDataDir)
	if userDataDir == "" {
		userDataDir = getUserDataDir()
	}

	logFile := strings.TrimSpace(args.LogFile)
	if logFile == "" {
		logFile = filepath.Join(os.TempDir(), "chrome-launch.log")
	}

	fmt.Printf("Launching Chrome on port %d...\n", defaultDebugPort)
	fmt.Printf("Chrome path: %s\n", chromePath)
	fmt.Printf("User data: %s\n", userDataDir)
	fmt.Printf("Logs: %s\n", logFile)

	lf, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening log file: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = lf.Close() }()

	chromeArgs := []string{
		fmt.Sprintf("--remote-debugging-port=%d", defaultDebugPort),
		"--remote-debugging-address=127.0.0.1",
		fmt.Sprintf("--user-data-dir=%s", userDataDir),
		"--no-first-run",
		"--no-default-browser-check",
	}

	cmd := exec.Command(chromePath, chromeArgs...)
	cmd.Stdout = lf
	cmd.Stderr = lf

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "error launching Chrome: %v\n", err)
		os.Exit(1)
	}
	if cmd.Process != nil {
		_ = cmd.Process.Release()
	}

	fmt.Printf("Waiting for Chrome to start")
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		fmt.Print(".")
		if lib.IsChromeRunning() {
			fmt.Println()
			fmt.Printf("Chrome is running on port %d\n", defaultDebugPort)
			return
		}
	}

	fmt.Println()
	fmt.Fprintf(os.Stderr, "warning: Chrome may still be starting. Check %s for details.\n", logFile)
}

// findChrome locates the Chrome executable based on the current platform.
func findChrome() string {
	// Check CHROME_PATH env var first.
	if p := strings.TrimSpace(os.Getenv("CHROME_PATH")); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	var candidates []string

	switch runtime.GOOS {
	case "darwin":
		candidates = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
		}
	case "linux":
		if isWSL() {
			candidates = []string{
				"/mnt/c/Program Files/Google/Chrome/Application/chrome.exe",
				"/mnt/c/Program Files (x86)/Google/Chrome/Application/chrome.exe",
			}
		}
		candidates = append(candidates,
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/snap/bin/chromium",
		)
	case "windows":
		candidates = []string{
			filepath.Join(os.Getenv("PROGRAMFILES"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("PROGRAMFILES(X86)"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Google", "Chrome", "Application", "chrome.exe"),
		}
	}

	for _, p := range candidates {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}

	for _, name := range []string{"google-chrome", "google-chrome-stable", "chromium", "chromium-browser", "chrome"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}

	return ""
}

// getUserDataDir returns the user data directory for Chrome.
func getUserDataDir() string {
	if isWSL() {
		return "C:\\temp\\chrome"
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "chrome")
	}
	return filepath.Join(home, ".chrome")
}

// isWSL detects if running under Windows Subsystem for Linux.
func isWSL() bool {
	if runtime.GOOS != "linux" {
		return false
	}

	if _, err := os.Stat("/proc/sys/fs/binfmt_misc/WSLInterop"); err == nil {
		return true
	}

	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}

	s := string(data)
	return strings.Contains(s, "Microsoft") || strings.Contains(s, "WSL")
}
