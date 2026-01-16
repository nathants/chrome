// title provides Chrome title fetching command
package title

import (
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/chromedp/chromedp"
	"github.com/nathants/chrome/lib"
)

func init() {
	lib.Commands["title"] = title
	lib.Args["title"] = titleArgs{}
}

type titleArgs struct {
	lib.TargetArgs
}

func (titleArgs) Description() string {
	return `title - Get page title

Prints the title of the current Chrome page.

Example:
  chrome title`
}

func title() {
	var args titleArgs
	arg.MustParse(&args)

	ctx, cancel := lib.SetupContext()
	defer cancel()

	targetCtx, targetCancel, err := lib.EnsureTargetContext(ctx, args.TargetArgs.Selector())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer targetCancel()

	var title string
	if err := chromedp.Run(targetCtx, chromedp.Title(&title)); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(title)
}