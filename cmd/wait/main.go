// wait provides Chrome wait-for-text command
package wait

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/chromedp/chromedp"
	"github.com/nathants/chrome/lib"
)

func init() {
	lib.Commands["wait"] = wait
	lib.Args["wait"] = waitArgs{}
}

type waitArgs struct {
	lib.TargetArgs
	Text    string `arg:"positional,required" help:"text to wait for"`
	Timeout int    `arg:"--timeout" default:"10" help:"timeout in seconds"`
}

func (waitArgs) Description() string {
	return `wait - Wait for text to appear

Waits for the specified text to appear on the page.

Example:
  chrome wait "Success"
  chrome wait "Loading complete" --timeout 30`
}

func wait() {
	var args waitArgs
	arg.MustParse(&args)

	ctx, cancel := lib.SetupContext()
	defer cancel()

	targetCtx, targetCancel, err := lib.EnsureTargetContext(ctx, args.TargetArgs.Selector())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer targetCancel()

	timeout := time.Duration(args.Timeout) * time.Second
	err = waitForText(targetCtx, args.Text, timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func waitForText(ctx context.Context, text string, timeout time.Duration) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	script := fmt.Sprintf(`
		(() => {
			const text = %s;
			return document.body.innerText.includes(text);
		})()
	`, strconv.Quote(text))

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("timeout waiting for text: %s", text)
		case <-ticker.C:
			var found bool
			err := chromedp.Run(timeoutCtx, chromedp.Evaluate(script, &found))
			if err != nil {
				return err
			}
			if found {
				return nil
			}
		}
	}
}
