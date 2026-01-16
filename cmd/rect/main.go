// rect provides element bounding rectangle command
package rect

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/chromedp/chromedp"
	"github.com/nathants/chrome/lib"
)

func init() {
	lib.Commands["rect"] = rect
	lib.Args["rect"] = rectArgs{}
}

type rectArgs struct {
	lib.TargetArgs
	Selector string `arg:"positional,required" help:"CSS selector of element"`
}

func (rectArgs) Description() string {
	return `rect - Get element bounding rectangle

Returns the position and size of an element in the viewport.
Useful for debugging layout issues and calculating click coordinates.

Example:
  chrome rect "canvas"
  chrome rect "#submit-button"`
}

func rect() {
	var args rectArgs
	arg.MustParse(&args)

	ctx, cancel := lib.SetupContext()
	defer cancel()

	targetCtx, targetCancel, err := lib.EnsureTargetContext(ctx, args.TargetArgs.Selector())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer targetCancel()

	var result map[string]interface{}

	script := fmt.Sprintf(`
		(function() {
			const el = document.querySelector(%q);
			if (!el) return null;
			const rect = el.getBoundingClientRect();
			return {
				x: rect.x,
				y: rect.y,
				width: rect.width,
				height: rect.height,
				top: rect.top,
				right: rect.right,
				bottom: rect.bottom,
				left: rect.left
			};
		})()
	`, args.Selector)

	err = chromedp.Run(targetCtx, chromedp.Evaluate(script, &result))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if result == nil {
		fmt.Fprintf(os.Stderr, "error: element not found: %s\n", args.Selector)
		os.Exit(1)
	}

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(jsonBytes))
}