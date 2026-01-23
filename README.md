# chrome

A command-line interface for Chrome automation via the Chrome DevTools Protocol (CDP).

## Installation

```bash
go install github.com/nathants/chrome@latest
```

Or build from source:

```bash
git clone https://github.com/nathants/chrome
cd chrome
go build
```

## Usage

There are two modes of operation:

### 1. External Chrome

Launch Chrome with remote debugging enabled, then run commands against it:

```bash
# Launch Chrome with debugging port
chrome launch

# Run commands
chrome navigate https://example.com
chrome screenshot
chrome click "button#submit"
chrome type "#email" "user@example.com"
```

### 2. Headless Mode

Commands automatically launch a headless Chrome instance for single operations:

```bash
chrome screenshot --path /tmp/shot.png
```

## Commands

| Command | Description |
|---------|-------------|
| `launch` | Launch Chrome with remote debugging |
| `instances` | List running Chrome instances |
| `navigate` | Navigate to a URL |
| `newtab` | Create a new tab |
| `close` | Close a tab |
| `list` | List open tabs |
| `click` | Click an element by CSS selector |
| `clicktext` | Click an element by its visible text |
| `clickxy` | Click at specific coordinates |
| `type` | Type text into an element |
| `eval` | Evaluate JavaScript |
| `fill` | Fill an input field |
| `wait` | Wait for text to appear |
| `waitfor` | Wait for an element to appear |
| `screenshot` | Capture a screenshot |
| `html` | Get page HTML |
| `title` | Get page title |
| `rect` | Get element bounding rectangle |
| `console` | Capture console logs |
| `network` | Monitor network requests |
| `step` | Run action + screenshot in one command |
| `slideshow` | Generate MP4 from captured steps |

Run `chrome <command> --help` for detailed usage of each command.

## Tab Targeting

When multiple tabs are open, use global `-t` to target a specific tab by URL prefix (recommended):

```bash
chrome -t http://localhost:3000 click "button.submit"
chrome -t https://example screenshot
```

You can also pass `-t/--target` after the command if you prefer:

```bash
chrome click -t http://localhost:3000 "button.submit"
```

Or set the `CHROME_TARGET` environment variable:

```bash
export CHROME_TARGET=http://localhost:3000
chrome click "button.submit"
```

## Multiple Instances

Run multiple Chrome instances simultaneously with different profiles on different ports:

```bash
# Launch instances on different ports with different profiles
chrome launch                                                 # Port 9222, default profile
chrome launch --port 9223 --user-data-dir ~/.chrome-twitter   # Port 9223, Twitter profile
chrome launch --port 9224 --user-data-dir ~/.chrome-github    # Port 9224, GitHub profile

# List running instances
chrome instances

# Target a specific instance with -p flag
chrome -p 9223 newtab https://x.com
chrome -p 9223 list

# Or use environment variable
export CHROME_PORT=9223
chrome list
```

Each instance has its own profile directory for persistent cookies/auth.

## Workflow: Step-by-Step Automation

The `step` command combines an action with an automatic screenshot, useful for documenting automation workflows.
Pass the action and its args as separate tokens:

```bash
chrome step navigate https://example.com
chrome step click "button.login"
chrome step type "#username" "alice"
chrome step --note "After login" click "button.submit"
```

If you have a single quoted action (for example `"click #btn"`), `step` will split it on whitespace.

Screenshots are saved to `~/chrome-shots/` by default with metadata JSON files.

Generate a video slideshow from captured steps:

```bash
chrome slideshow
chrome slideshow --verbose
```

By default ffmpeg output is quiet; use `--verbose` to show banner and progress.

## DevTools: Console and Network

### Console Logs

Capture browser console output including `console.log`, `console.warn`, `console.error`,
JavaScript exceptions, and browser log events (CSP violations, security errors, deprecation warnings).

```bash
# Capture for 5 seconds (default)
chrome console

# Capture for 10 seconds
chrome console -d 10

# Run JavaScript after capture starts (useful for triggering logs)
chrome console --eval "document.querySelector('#emit-logs').click()" -d 2

# Follow mode (continuous until Ctrl+C)
chrome console -f
```

Output is JSON, one object per line:

```json
{"type": "log", "message": "Hello world", "timestamp": "..."}
{"type": "warning", "message": "Deprecated API", "timestamp": "..."}
{"type": "error", "message": "Something failed", "timestamp": "..."}
{"type": "exception", "message": "Error: ...", "level": "error", "timestamp": "..."}
{"type": "security", "message": "CSP violation...", "level": "error", "timestamp": "..."}
```

The `type` field indicates the source:
- `log`, `warning`, `error`, `info`, `debug` - console API calls
- `exception` - uncaught JavaScript exceptions
- `security` - CSP violations, mixed content warnings
- `deprecation` - deprecated API usage
- `network` - network-related errors
- `violation` - performance violations

### Network Requests

Monitor HTTP requests and responses:

```bash
# Monitor for 5 seconds (default)
chrome network

# Monitor for 10 seconds
chrome network -d 10

# Run JavaScript after capture starts (useful for triggering requests)
chrome network --eval "fetch('/data.json')" -d 2

# Follow mode (continuous until Ctrl+C)
chrome network -f
```

Output is JSON, one object per line:

```json
{"type": "request", "requestId": "123", "url": "https://api.example.com/data", "method": "GET", "timestamp": "..."}
{"type": "response", "requestId": "123", "url": "https://api.example.com/data", "status": 200, "statusText": "OK", "timestamp": "..."}
{"type": "failed", "requestId": "456", "timestamp": "..."}
```

### Typical Workflow

Run console/network monitoring in the background while interacting with the page:

```bash
# Terminal 1: Start monitoring
export CHROME_TARGET=http://localhost:3000
chrome console -f > /tmp/console.log &

# Terminal 2: Interact with page
chrome click "#login-button"
chrome type "#username" "alice"
chrome click "#submit"

# View captured logs
cat /tmp/console.log
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `CHROME_PORT` | Chrome debug port (default: 9222) |
| `CHROME_TARGET` | Default tab URL prefix for targeting |
| `CHROME_PATH` | Path to Chrome executable |

## Security Notes

Chrome remote debugging (CDP) is powerful: anyone who can access the debugging endpoint can
control your browser, read page content, and execute JavaScript.

- `chrome launch` binds the debugging endpoint to `127.0.0.1`.
- Do not expose the remote debugging port on untrusted networks.
- The `launch` command uses a user-data-dir (`~/.chrome` on Unix by default) which may
  persist cookies/session state.
- If Chrome is already running, the profile may be locked; in that case, use
  `chrome launch --user-data-dir ~/.chrome` to isolate automation.

## Requirements

- Go
- Chrome
- `ffmpeg` (for slideshow generation)

## Platform Support

- Linux
- macOS
- Windows (WSL)
