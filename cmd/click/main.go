// click provides Chrome element clicking command
package click

import (
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/chromedp/chromedp"
	"github.com/nathants/chrome/lib"
)

func init() {
	lib.Commands["click"] = click
	lib.Args["click"] = clickArgs{}
}

type clickArgs struct {
	lib.TargetArgs
	Selector string `arg:"positional,required" help:"CSS selector of element to click"`
}

func (clickArgs) Description() string {
	return `click - Click an element by CSS selector

Uses Chrome DevTools Protocol to send a real mouse click to an element.
This properly triggers React event handlers and works with canvas elements.

Example:
  chrome click "button#submit"
  chrome click ".object-button:nth-child(2)"
  chrome click "canvas"`
}

func click() {
	var args clickArgs
	arg.MustParse(&args)

	ctx, cancel := lib.SetupContext()
	defer cancel()

	targetCtx, targetCancel, err := lib.EnsureTargetContext(ctx, args.TargetArgs.Selector())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer targetCancel()

	err = chromedp.Run(targetCtx, chromedp.Click(args.Selector, chromedp.ByQuery))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}