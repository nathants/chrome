// newtab provides Chrome tab creation command
package newtab

import (
	"context"
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/nathants/chrome/lib"
)

func init() {
	lib.Commands["newtab"] = newtab
	lib.Args["newtab"] = newtabArgs{}
}

type newtabArgs struct {
	URL string `arg:"positional" default:"about:blank" help:"URL to open in new tab"`
}

func (newtabArgs) Description() string {
	return `newtab - Create a new tab

Creates a new browser tab and optionally navigates to a URL.

Example:
  chrome newtab                           # Create blank tab
  chrome newtab http://localhost:8000     # Create tab and navigate`
}

func newtab() {
	var args newtabArgs
	arg.MustParse(&args)

	if !lib.IsChromeRunning() {
		fmt.Fprintf(os.Stderr, "Chrome not running on port %d\n", lib.GetPort())
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), lib.DefaultTimeout)
	defer cancel()

	allocCtx, allocCancel := chromedp.NewRemoteAllocator(ctx, lib.ChromeURL())
	defer allocCancel()

	// Create new context - this will create a new tab on first Run
	// DO NOT cancel this context - we want to leave the tab open
	tabCtx, _ := chromedp.NewContext(allocCtx)

	// Get target ID and optionally navigate
	var targetID target.ID
	actions := []chromedp.Action{
		chromedp.ActionFunc(func(ctx context.Context) error {
			targetID = chromedp.FromContext(ctx).Target.TargetID
			return nil
		}),
	}

	// Add navigation if URL provided
	if args.URL != "about:blank" {
		actions = append(actions, chromedp.Navigate(args.URL))
	}

	if err := chromedp.Run(tabCtx, actions...); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created tab: %s\n", targetID)
}