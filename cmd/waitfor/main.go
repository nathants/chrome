// waitfor provides Chrome element waiting command
package waitfor

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/chromedp/chromedp"
	"github.com/nathants/chrome/lib"
)

func init() {
	lib.Commands["waitfor"] = waitfor
	lib.Args["waitfor"] = waitforArgs{}
}

type waitforArgs struct {
	lib.TargetArgs
	Selector string `arg:"positional,required" help:"CSS selector to wait for"`
	Timeout  int    `arg:"--timeout" default:"10" help:"timeout in seconds"`
}

func (waitforArgs) Description() string {
	return `waitfor - Wait for an element to appear

Waits for an element matching the CSS selector to become visible in the DOM.
Useful for waiting for dynamic content, AJAX responses, or React state updates.

Example:
  chrome waitfor "#results"
  chrome waitfor ".loading-complete" --timeout 30
  chrome waitfor "button:not([disabled])"`
}

func waitfor() {
	var args waitforArgs
	arg.MustParse(&args)

	ctx, cancel := lib.SetupContext()
	defer cancel()

	targetCtx, targetCancel, err := lib.EnsureTargetContext(ctx, args.TargetArgs.Selector())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer targetCancel()

	if err := waitForVisible(targetCtx, args.Selector, time.Duration(args.Timeout)*time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// waitForVisible waits until the element exists in the DOM and appears visible (non-zero box and not hidden)
func waitForVisible(ctx context.Context, sel string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	check := fmt.Sprintf(`(() => {
  const el = document.querySelector(%q);
  if (!el) {
    return { ok: false, reason: 'missing' };
  }
  const rect = el.getBoundingClientRect();
  const style = getComputedStyle(el);
  const visible = rect.width > 0 && rect.height > 0 && style.display !== 'none' && style.visibility !== 'hidden' && parseFloat(style.opacity || '1') > 0;
  return { ok: visible };
})()`, sel)

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for visible: %s", sel)
		case <-ticker.C:
			var result struct {
				Ok bool `json:"ok"`
			}
			if err := chromedp.Run(ctx, chromedp.Evaluate(check, &result)); err != nil {
				continue
			}
			if result.Ok {
				return nil
			}
		}
	}
}
