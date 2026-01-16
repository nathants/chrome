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
| `navigate` | Navigate to a URL |
| `newtab` | Create a new tab |
| `close` | Close a tab |
| `list` | List open tabs |
| `click` | Click an element by CSS selector |
| `clicktext` | Click an element by its visible text |
| `clickxy` | Click at specific coordinates |
| `type` | Type text into an element |
| `eval` | Evaluate JavaScript |
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

When multiple tabs are open, use `-t` to target a specific tab by URL prefix:

```bash
chrome -t http://localhost:3000 click "button.submit"
chrome -t https://example screenshot
```

Or set the `CHROME_TARGET` environment variable:

```bash
export CHROME_TARGET=http://localhost:3000
chrome click "button.submit"
```

## Workflow: Step-by-Step Automation

The `step` command combines an action with an automatic screenshot, useful for documenting automation workflows:

```bash
chrome step navigate https://example.com
chrome step click "button.login"
chrome step type "#username" "alice"
chrome step --note "After login" click "button.submit"
```

Screenshots are saved to `~/chrome-shots/` by default with metadata JSON files.

Generate a video slideshow from captured steps:

```bash
chrome slideshow
```

## Environment Variables

| Variable | Description |
|----------|-------------|
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
