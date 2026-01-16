// clickxy provides Chrome coordinate clicking command
package clickxy

import (
	"fmt"
	"os"
	"strconv"

	"github.com/alexflint/go-arg"
	"github.com/chromedp/chromedp"
	"github.com/nathants/chrome/lib"
)

func init() {
	lib.Commands["clickxy"] = clickxy
	lib.Args["clickxy"] = clickxyArgs{}
}

type clickxyArgs struct {
	lib.TargetArgs
	X string `arg:"positional,required" help:"X coordinate in pixels"`
	Y string `arg:"positional,required" help:"Y coordinate in pixels"`
}

func (clickxyArgs) Description() string {
	return `clickxy - Click at specific screen coordinates

Sends a real mouse click event at the specified X, Y coordinates.
Coordinates are viewport-relative (not page-relative).

Example:
  chrome clickxy 300 200
  chrome clickxy -t "test page" 100 150`
}

func clickxy() {
	var args clickxyArgs
	arg.MustParse(&args)

	x, err := strconv.ParseFloat(args.X, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid x coordinate: %v\n", err)
		os.Exit(1)
	}

	y, err := strconv.ParseFloat(args.Y, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid y coordinate: %v\n", err)
		os.Exit(1)
	}

	ctx, cancel := lib.SetupContext()
	defer cancel()

	targetCtx, targetCancel, err := lib.EnsureTargetContext(ctx, args.TargetArgs.Selector())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer targetCancel()

	err = chromedp.Run(targetCtx, chromedp.MouseClickXY(x, y))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}