// eval provides Chrome JavaScript evaluation command
package eval

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/chromedp/chromedp"
	"github.com/nathants/chrome/lib"
)

func init() {
	lib.Commands["eval"] = eval
	lib.Args["eval"] = evalArgs{}
}

type evalArgs struct {
	lib.TargetArgs
	Script string `arg:"positional,required" help:"JavaScript code to evaluate"`
}

func (evalArgs) Description() string {
	return `eval - Evaluate JavaScript in Chrome

Evaluates JavaScript code in the current Chrome page and prints the result.

Example:
  chrome eval "document.title"
  chrome eval "document.getElementById('nameInput').value = 'test'"`
}

func eval() {
	var args evalArgs
	arg.MustParse(&args)

	ctx, cancel := lib.SetupContext()
	defer cancel()

	targetCtx, targetCancel, err := lib.EnsureTargetContext(ctx, args.TargetArgs.Selector())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer targetCancel()

	var result interface{}

	err = chromedp.Run(targetCtx, chromedp.Evaluate(args.Script, &result))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	switch v := result.(type) {
	case string:
		fmt.Println(v)
	case nil:
		// No output for undefined/null
	default:
		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(jsonBytes))
	}
}