// quit provides command to quit a Chrome instance
package quit

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"
	"github.com/nathants/chrome/lib"
)

func init() {
	lib.Commands["quit"] = quitChrome
	lib.Args["quit"] = quitArgs{}
}

type quitArgs struct{}

func (quitArgs) Description() string {
	return `quit - Quit Chrome instance

Gracefully closes the Chrome instance running on the current port.
Uses Chrome DevTools Protocol to send a close command.

The port is determined by:
  1. -p/--port flag (before the command)
  2. CHROME_PORT environment variable
  3. Default: 9222

Example:
  chrome quit                    # Quit Chrome on default port 9222
  chrome -p 9223 quit            # Quit Chrome on port 9223
  CHROME_PORT=9223 chrome quit   # Same as above`
}

func quitChrome() {
	var args quitArgs
	arg.MustParse(&args)

	port := lib.GetPort()

	if !lib.IsChromeRunningOnPort(port) {
		fmt.Printf("No Chrome instance running on port %d\n", port)
		os.Exit(0)
	}

	fmt.Printf("Quitting Chrome on port %d...\n", port)

	// Create a context to connect to Chrome
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	chromeURL := fmt.Sprintf("http://localhost:%d", port)
	allocCtx, allocCancel := chromedp.NewRemoteAllocator(ctx, chromeURL)
	defer allocCancel()

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	// Send Browser.close command to gracefully close Chrome
	err := chromedp.Run(browserCtx, browser.Close())
	if err != nil {
		// If the close command fails, Chrome may have already closed
		// or there was a connection issue - either way, check if it's gone
		time.Sleep(500 * time.Millisecond)
		if lib.IsChromeRunningOnPort(port) {
			fmt.Fprintf(os.Stderr, "error: failed to quit Chrome: %v\n", err)
			os.Exit(1)
		}
	}

	// Wait a moment and verify Chrome has stopped
	time.Sleep(500 * time.Millisecond)
	if lib.IsChromeRunningOnPort(port) {
		fmt.Fprintf(os.Stderr, "warning: Chrome may still be running on port %d\n", port)
		os.Exit(1)
	}

	// Clean up instance metadata
	_ = lib.RemoveInstanceMetadata(port)

	fmt.Printf("Chrome on port %d has been closed\n", port)
}
