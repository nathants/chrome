// type provides Chrome typing command
package typecommand

import (
	"fmt"
	"os"
	"strconv"

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
	Append   bool   `arg:"--append,-a" help:"Append to existing text instead of replacing"`
}

func (typeArgs) Description() string {
	return `type - Type text into an element

Clears the field first (select all + delete), then types the text.
This ensures predictable behavior with React controlled inputs.

Use --append to add to existing text instead of replacing.

Example:
  chrome type "#nameInput" "Alice"
  chrome type "input[name='email']" "alice@test.com"
  chrome type --append "textarea" " more text"`
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

	var actions []chromedp.Action
	actions = append(actions, chromedp.Focus(args.Selector, chromedp.ByQuery))
	if !args.Append {
		// Select all existing text within the element so new text replaces it
		// For INPUT/TEXTAREA: use element.select()
		// For contenteditable: use Selection API to select all content
		selectScript := `(() => {
		  const el = document.querySelector(` + strconv.Quote(args.Selector) + `);
		  if (!el) return;
		  if (typeof el.select === 'function') {
		    el.select();
		  } else if (el.isContentEditable) {
		    const range = document.createRange();
		    range.selectNodeContents(el);
		    const sel = window.getSelection();
		    sel.removeAllRanges();
		    sel.addRange(range);
		  }
		})()`
		actions = append(actions, chromedp.Evaluate(selectScript, nil))
	}
	actions = append(actions, chromedp.SendKeys(args.Selector, args.Text, chromedp.ByQuery))

	err = chromedp.Run(targetCtx, actions...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}