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
	_ "github.com/nathants/chrome/cmd/fill"
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
	fmt.Fprintln(os.Stderr, "Chrome CLI - Browser automation for LLM agents")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Quick Start:")
	fmt.Fprintln(os.Stderr, "  chrome list                              # See open tabs (use this first!)")
	fmt.Fprintln(os.Stderr, "  chrome newtab http://localhost:8000      # Open a new tab")
	fmt.Fprintln(os.Stderr, "  chrome -t localhost:8000 click \"#btn\"    # Target tab by URL prefix")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Global Options (must appear before command):")
	fmt.Fprintln(os.Stderr, "  -t, --target URL_PREFIX                  # Select tab by URL prefix")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Multi-Agent Usage:")
	fmt.Fprintln(os.Stderr, "  Multiple agents share the same Chrome instance on port 9222.")
	fmt.Fprintln(os.Stderr, "  - Run `chrome list` to see existing tabs before creating new ones")
	fmt.Fprintln(os.Stderr, "  - Use `chrome -t URL_PREFIX <cmd>` to target your specific tab")
	fmt.Fprintln(os.Stderr, "  - NEVER kill Chrome processes - other agents may be using them")
	fmt.Fprintln(os.Stderr, "  - Use `chrome launch` only if `chrome list` fails (Chrome not running)")
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
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	args := append([]string{}, os.Args[1:]...)
	target := ""
	for len(args) > 0 {
		arg := args[0]
		if arg == "-h" || arg == "--help" {
			usage()
			os.Exit(0)
		}
		if arg == "-t" || arg == "--target" {
			if len(args) < 2 {
				fmt.Fprintln(os.Stderr, "error: --target requires a value")
				os.Exit(1)
			}
			target = args[1]
			args = args[2:]
			continue
		}
		if strings.HasPrefix(arg, "--target=") {
			target = strings.TrimPrefix(arg, "--target=")
			args = args[1:]
			continue
		}
		if strings.HasPrefix(arg, "-t=") {
			target = strings.TrimPrefix(arg, "-t=")
			args = args[1:]
			continue
		}
		break
	}
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}
	if strings.TrimSpace(target) != "" {
		err := os.Setenv("CHROME_TARGET", target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}

	cmd := args[0]
	fn, ok := lib.Commands[cmd]
	if !ok {
		usage()
		fmt.Fprintln(os.Stderr, "\nunknown command:", cmd)
		os.Exit(1)
	}
	os.Args = args
	fn()
}
