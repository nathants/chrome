// main provides CLI entry point for chrome with subcommand routing
package main

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/alexflint/go-arg"
	_ "github.com/nathants/chrome/cmd/click"
	_ "github.com/nathants/chrome/cmd/clicktext"
	_ "github.com/nathants/chrome/cmd/clickxy"
	_ "github.com/nathants/chrome/cmd/close"
	_ "github.com/nathants/chrome/cmd/console"
	_ "github.com/nathants/chrome/cmd/eval"
	_ "github.com/nathants/chrome/cmd/html"
	_ "github.com/nathants/chrome/cmd/launch"
	_ "github.com/nathants/chrome/cmd/list"
	_ "github.com/nathants/chrome/cmd/navigate"
	_ "github.com/nathants/chrome/cmd/network"
	_ "github.com/nathants/chrome/cmd/newtab"
	_ "github.com/nathants/chrome/cmd/rect"
	_ "github.com/nathants/chrome/cmd/screenshot"
	_ "github.com/nathants/chrome/cmd/slideshow"
	_ "github.com/nathants/chrome/cmd/step"
	_ "github.com/nathants/chrome/cmd/title"
	_ "github.com/nathants/chrome/cmd/type"
	_ "github.com/nathants/chrome/cmd/wait"
	_ "github.com/nathants/chrome/cmd/waitfor"
	"github.com/nathants/chrome/lib"
)

func usage() {
	fmt.Fprintln(os.Stderr, "Chrome CLI - Simple CDP interface")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Modes:")
	fmt.Fprintln(os.Stderr, "  1. External Chrome: Launch Chrome with --remote-debugging-port=9222 via `chrome launch`")
	fmt.Fprintln(os.Stderr, "  2. Headless: CLI launches headless Chrome automatically")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Selectors:")
	fmt.Fprintln(os.Stderr, "  Commands that take SELECTOR use standard CSS selectors only.")
	fmt.Fprintln(os.Stderr, "  Valid:   #id, .class, button, input[type=\"email\"], div > span")
	fmt.Fprintln(os.Stderr, "  Invalid: :has-text(), :text(), >> (these are Playwright selectors, not CSS)")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "  To click by visible text, use 'clicktext' instead of 'click':")
	fmt.Fprintln(os.Stderr, "    chrome clicktext \"Sign In\"     # correct - clicks button with text 'Sign In'")
	fmt.Fprintln(os.Stderr, "    chrome click \"button#submit\"   # correct - clicks button with id 'submit'")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")

	var fns []string
	maxLen := 0
	for fn := range lib.Commands {
		fns = append(fns, fn)
		if len(fn) > maxLen {
			maxLen = len(fn)
		}
	}
	sort.Strings(fns)
	fmtStr := "  %-" + fmt.Sprint(maxLen) + "s %s\n"
	for _, fn := range fns {
		args := lib.Args[fn]
		val := reflect.ValueOf(args)
		newVal := reflect.New(val.Type())
		newVal.Elem().Set(val)
		p, err := arg.NewParser(arg.Config{}, newVal.Interface())
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error creating parser:", err)
			return
		}
		var buffer bytes.Buffer
		p.WriteHelp(&buffer)
		descr := buffer.String()
		lines := strings.Split(descr, "\n")
		var line string
		for _, l := range lines {
			l = strings.TrimSpace(l)
			if strings.HasPrefix(l, "Usage:") {
				line = l
			}
		}
		line = strings.ReplaceAll(line, "Usage: chrome", "")
		fmt.Fprintf(os.Stderr, fmtStr, fn, line)
	}
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Note: For multi-step automation, use external Chrome on port 9222 to maintain state between commands.")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Workflow tip: 'chrome step <action>' wraps any action with a screenshot.")
	fmt.Fprintln(os.Stderr, "  Step runs the action, then takes its own screenshot. Flags like --output-dir")
	fmt.Fprintln(os.Stderr, "  control where step saves its screenshot, not the action itself.")
}

func main() {
	if len(os.Args) < 2 || os.Args[1] == "-h" || os.Args[1] == "--help" {
		usage()
		os.Exit(1)
	}
	cmd := os.Args[1]
	fn, ok := lib.Commands[cmd]
	if !ok {
		usage()
		fmt.Fprintln(os.Stderr, "\nunknown command:", cmd)
		os.Exit(1)
	}
	os.Args = os.Args[1:]
	fn()
}
