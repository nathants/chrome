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
	Port        int    `arg:"-p,--port" default:"9222" help:"Debug port (default: 9222)"`
	UserDataDir string `arg:"--user-data-dir" help:"Chrome user data dir (default: ~/.chrome on Unix, C:\\temp\\chrome on WSL)"`
}

func (launchArgs) Description() string {
	return `launch - Launch Chrome with remote debugging

Launches Chrome with remote debugging enabled.

Security: remote debugging grants full control of Chrome. This command binds the
debugging endpoint to localhost only.

Profiles: Use --user-data-dir to persist cookies/auth across sessions. The profile
directory stores cookies, localStorage, and other browser state.

Multiple Instances: Use --port to run multiple Chrome instances simultaneously,
each with its own profile. Use 'chrome instances' to list running instances.

Defaults:
  Port:        9222
  Linux/macOS: ~/.chrome
  WSL:         C:\temp\chrome

Logs: Written to <tmp>/chrome-launch-<profile>.log based on profile directory name.
  Linux/macOS: /tmp/chrome-launch-chrome.log (default profile)
               /tmp/chrome-launch-myprofile.log (--user-data-dir ~/.chrome-myprofile)
  WSL:         /tmp/chrome-launch-chrome.log (default profile)
               /tmp/chrome-launch-myprofile.log (--user-data-dir C:\temp\chrome-myprofile)

WSL Note: On WSL, Chrome runs on Windows, so --user-data-dir must be a Windows path.
  Correct: C:\temp\chrome-myprofile
  Wrong:   ~/.chrome-myprofile (Linux paths don't work)

Example:
  chrome launch
  chrome launch --port 9223 --user-data-dir ~/.chrome-twitter
  chrome launch --user-data-dir ~/.chrome-myprofile           # Linux/macOS
  chrome launch --user-data-dir 'C:\temp\chrome-myprofile'    # WSL`
}

func launchChrome() {
	var args launchArgs
	arg.MustParse(&args)

	port := args.Port
	if lib.IsChromeRunningOnPort(port) {
		fmt.Fprintf(os.Stderr, "Chrome already running on port %d. Use:\n", port)
		fmt.Fprintf(os.Stderr, "  chrome -p %d list              # See open tabs\n", port)
		fmt.Fprintf(os.Stderr, "  chrome -p %d newtab <url>      # Open new tab\n", port)
		fmt.Fprintf(os.Stderr, "  chrome -p %d -t <url> <cmd>    # Target existing tab\n", port)
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

	// Log file is derived from user data dir name
	profileName := filepath.Base(userDataDir)
	// Handle Windows paths on WSL (e.g., C:\temp\chrome-myprofile -> chrome-myprofile)
	if idx := strings.LastIndex(profileName, "\\"); idx >= 0 {
		profileName = profileName[idx+1:]
	}
	logFile := filepath.Join(os.TempDir(), fmt.Sprintf("chrome-launch-%s.log", profileName))

	fmt.Printf("Launching Chrome on port %d...\n", port)
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
		fmt.Sprintf("--remote-debugging-port=%d", port),
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

	pid := 0
	if cmd.Process != nil {
		pid = cmd.Process.Pid
		_ = cmd.Process.Release()
	}

	fmt.Printf("Waiting for Chrome to start")
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		fmt.Print(".")
		if lib.IsChromeRunningOnPort(port) {
			fmt.Println()
			fmt.Printf("Chrome is running on port %d\n", port)

			// Write instance metadata
			info := lib.InstanceInfo{
				Port:        port,
				UserDataDir: userDataDir,
				PID:         pid,
				StartedAt:   time.Now().Format(time.RFC3339),
			}
			if err := lib.WriteInstanceMetadata(info); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to write instance metadata: %v\n", err)
			}
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
