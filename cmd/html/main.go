// html provides Chrome HTML dumping command
package html

import (
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/chromedp/chromedp"
	"github.com/nathants/chrome/lib"
)

func init() {
	lib.Commands["html"] = html
	lib.Args["html"] = htmlArgs{}
}

type htmlArgs struct {
	lib.TargetArgs
	Outer bool `arg:"-o,--outer" help:"return outerHTML instead of documentElement.innerHTML"`
}

func (htmlArgs) Description() string {
	return `html - Get page HTML

Prints the HTML of the current Chrome page.

Example:
  chrome html
  chrome html -o`
}

func html() {
	var args htmlArgs
	arg.MustParse(&args)

	ctx, cancel := lib.SetupContext()
	defer cancel()

	targetCtx, targetCancel, err := lib.EnsureTargetContext(ctx, args.TargetArgs.Selector())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer targetCancel()

	script := "document.documentElement.innerHTML"
	if args.Outer {
		script = "document.documentElement.outerHTML"
	}

	var html string
	if err := chromedp.Run(targetCtx, chromedp.Evaluate(script, &html)); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(html)
}