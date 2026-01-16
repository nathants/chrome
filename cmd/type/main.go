// type provides Chrome typing command
package typecommand

import (
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/chromedp/chromedp"
	"github.com/nathants/chrome/lib"
)

func init() {
	lib.Commands["type"] = typeText
	lib.Args["type"] = typeArgs{}
}

type typeArgs struct {
	lib.TargetArgs
	Selector string `arg:"positional,required" help:"CSS selector of element to type into"`
	Text     string `arg:"positional,required" help:"Text to type"`
}

func (typeArgs) Description() string {
	return `type - Type text into an element

Uses Chrome DevTools Protocol to send real keyboard events to an element.
This properly triggers React onChange handlers and works with controlled inputs.

Example:
  chrome type "#nameInput" "Alice"
  chrome type "input[name='email']" "alice@test.com"
  chrome type "textarea" "Multi-line\ntext"`
}

func typeText() {
	var args typeArgs
	arg.MustParse(&args)

	ctx, cancel := lib.SetupContext()
	defer cancel()

	targetCtx, targetCancel, err := lib.EnsureTargetContext(ctx, args.TargetArgs.Selector())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer targetCancel()

	err = chromedp.Run(targetCtx, chromedp.SendKeys(args.Selector, args.Text, chromedp.ByQuery))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}