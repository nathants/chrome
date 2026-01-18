// clicktext provides Chrome click-by-text command
package clicktext

import (
	"fmt"
	"os"
	"strconv"

	"github.com/alexflint/go-arg"
	"github.com/chromedp/chromedp"
	"github.com/nathants/chrome/lib"
)

func init() {
	lib.Commands["clicktext"] = clicktext
	lib.Args["clicktext"] = clickTextArgs{}
}

type clickTextArgs struct {
	lib.TargetArgs
	Text     string `arg:"positional,required" help:"exact button/link text to click"`
	Selector string `arg:"--selector" default:"button, a, [role='button']" help:"CSS selector to limit search domain"`
	Index    int    `arg:"--index" default:"0" help:"if multiple matches, which one to click (0-based)"`
}

func (clickTextArgs) Description() string {
	return `clicktext - Click an element by its visible text

Finds the Nth element matching --selector whose textContent.trim() equals TEXT and clicks it.
Defaults to buttons and links. Use this instead of 'click' when you want to match by text.

Examples:
  chrome clicktext "Sign In"                    # click button/link with text "Sign In"
  chrome clicktext "Submit" --index 1           # click the second "Submit" button
  chrome clicktext "Save" --selector "button"   # only match buttons, not links

This is the correct way to click by text. Do NOT use 'click' with Playwright selectors:
  chrome clicktext "Login"                          # CORRECT
  chrome click "button:has-text(\"Login\")"         # WRONG - invalid CSS selector`
}

func clicktext() {
	var args clickTextArgs
	arg.MustParse(&args)

	ctx, cancel := lib.SetupContext()
	defer cancel()

	targetCtx, targetCancel, err := lib.EnsureTargetContext(ctx, args.TargetArgs.Selector())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer targetCancel()

	script := `(() => {
	  const sel = ` + strconv.Quote(args.Selector) + `;
	  const want = ` + strconv.Quote(args.Text) + `;
	  const idx = ` + strconv.Itoa(args.Index) + `;
	  const nodes = Array.from(document.querySelectorAll(sel));
	  const matches = nodes.filter(n => (n.textContent || '').trim() === want);
	  const el = matches[idx];
	  if (!el) return { ok: false, count: matches.length };
	  const rect = el.getBoundingClientRect();
	  el.scrollIntoView({block:'center', inline:'center'});
	  el.click();
	  return { ok: true, count: matches.length, x: rect.left + rect.width/2, y: rect.top + rect.height/2 };
	})()`

	type result struct {
		Ok    bool    `json:"ok"`
		Count int     `json:"count"`
		X     float64 `json:"x"`
		Y     float64 `json:"y"`
	}
	var res result
	if err := chromedp.Run(targetCtx, chromedp.Evaluate(script, &res)); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if !res.Ok {
		fmt.Fprintf(os.Stderr, "error: no element with text %q (selector %q), matches=%d\n", args.Text, args.Selector, res.Count)
		os.Exit(1)
	}
}