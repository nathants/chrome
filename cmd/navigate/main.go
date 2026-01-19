// navigate provides Chrome navigation command
package navigate

import (
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/chromedp/chromedp"
	"github.com/nathants/chrome/lib"
)

func init() {
	lib.Commands["navigate"] = navigate
	lib.Args["navigate"] = navigateArgs{}
}

type navigateArgs struct {
	lib.TargetArgs
	URL string `arg:"positional,required" help:"URL to navigate to"`
}

func (navigateArgs) Description() string {
	return `navigate - Navigate to a URL

Navigates the Chrome browser to the specified URL.

Example:
  chrome navigate http://localhost:8000
  chrome navigate https://example.com`
}

func navigate() {
	var args navigateArgs
	arg.MustParse(&args)

	ctx, cancel := lib.SetupContext()
	defer cancel()

	targetCtx, targetCancel, err := lib.EnsureTargetContext(ctx, args.TargetArgs.Selector())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer targetCancel()

	if err := chromedp.Run(targetCtx, chromedp.Navigate(args.URL)); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
