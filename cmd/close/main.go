// close provides Chrome tab closing command
package close

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/nathants/chrome/lib"
)

func init() {
	lib.Commands["close"] = closeTab
	lib.Args["close"] = closeArgs{}
}

type closeArgs struct {
	lib.TargetArgs
}

func (closeArgs) Description() string {
	return `close - Close a tab

Closes the specified Chrome tab.

Example:
  chrome close -t http://example.com   # Close tab with URL starting with http://example.com`
}

func closeTab() {
	var args closeArgs
	arg.MustParse(&args)

	if !lib.IsChromeRunning() {
		fmt.Fprintf(os.Stderr, "error: Chrome not running on port 9222\n")
		os.Exit(1)
	}

	targetID, _, err := lib.ResolveTargetWithArgs(args.TargetArgs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if targetID == "" {
		fmt.Fprintf(os.Stderr, "error: no target found\n")
		os.Exit(1)
	}

	// Close tab via HTTP endpoint (simpler than chromedp context)
	url := fmt.Sprintf("http://localhost:9222/json/close/%s", targetID)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Fprintf(os.Stderr, "error: status %d: %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	fmt.Printf("Closed tab: %s\n", targetID)
}
