// fill provides Chrome form filling command optimized for React
package fill

import (
	"fmt"
	"os"
	"strconv"

	"github.com/alexflint/go-arg"
	"github.com/chromedp/chromedp"
	"github.com/nathants/chrome/lib"
)

func init() {
	lib.Commands["fill"] = fill
	lib.Args["fill"] = fillArgs{}
}

type fillArgs struct {
	lib.TargetArgs
	Selector string `arg:"positional,required" help:"CSS selector of input element"`
	Value    string `arg:"positional,required" help:"Value to set"`
}

func (fillArgs) Description() string {
	return `fill - Fill a form field

Sets the value via JavaScript and dispatches input/change events.
This is the most reliable way to fill forms in React applications.

Unlike 'type' which simulates keystrokes, 'fill' directly sets the value
and triggers the events React needs to update its state.

Supports INPUT, TEXTAREA, SELECT, and contenteditable elements.

Example:
  chrome fill "#email" "user@example.com"
  chrome fill "input[name='password']" "secret123"
  chrome fill "textarea" "Hello world"
  chrome fill "#country" "us"
  chrome fill "[contenteditable]" "Rich text content"`
}

func fill() {
	var args fillArgs
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
	  const val = ` + strconv.Quote(args.Value) + `;
	  const el = document.querySelector(sel);
	  if (!el) return { ok: false, error: "element not found" };
	  
	  // Handle SELECT elements (no native setter trick needed)
	  if (el.tagName === 'SELECT') {
	    el.focus();
	    el.value = val;
	    el.dispatchEvent(new Event('change', { bubbles: true }));
	    return { ok: true, value: el.value };
	  }
	  
	  // Handle contenteditable elements
	  if (el.isContentEditable) {
	    el.focus();
	    el.textContent = val;
	    el.dispatchEvent(new InputEvent('input', {
	      bubbles: true,
	      cancelable: true,
	      inputType: 'insertText',
	      data: val,
	    }));
	    el.dispatchEvent(new Event('change', { bubbles: true }));
	    return { ok: true, value: el.textContent };
	  }
	  
	  // Validate element type for INPUT/TEXTAREA
	  if (el.tagName !== 'INPUT' && el.tagName !== 'TEXTAREA') return { ok: false, error: "fill only supports INPUT, TEXTAREA, SELECT, and contenteditable elements" };
	  
	  // Focus the element
	  el.focus();
	  
	  // For React 16+, we need to use the native setter and InputEvent
	  // This is the most reliable way to update controlled inputs
	  const proto = el.tagName === 'TEXTAREA' 
	    ? HTMLTextAreaElement.prototype 
	    : HTMLInputElement.prototype;
	  const setter = Object.getOwnPropertyDescriptor(proto, 'value').set;
	  setter.call(el, val);
	  
	  // Dispatch InputEvent (not Event) - this is what browsers actually fire
	  // React 17+ specifically listens for this
	  const inputEvent = new InputEvent('input', {
	    bubbles: true,
	    cancelable: true,
	    inputType: 'insertText',
	    data: val,
	  });
	  el.dispatchEvent(inputEvent);
	  
	  // Also dispatch change event for completeness
	  el.dispatchEvent(new Event('change', { bubbles: true }));
	  
	  return { ok: true, value: el.value };
	})()`

	type result struct {
		Ok    bool   `json:"ok"`
		Value string `json:"value"`
		Error string `json:"error"`
	}
	var res result
	if err := chromedp.Run(targetCtx, chromedp.Evaluate(script, &res)); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if !res.Ok {
		fmt.Fprintf(os.Stderr, "error: %s (selector %q)\n", res.Error, args.Selector)
		os.Exit(1)
	}

	// Verify the value was set correctly
	if res.Value != args.Value {
		fmt.Fprintf(os.Stderr, "warning: value mismatch - requested %q but got %q\n", args.Value, res.Value)
	}
}
