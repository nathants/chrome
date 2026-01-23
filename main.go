// main provides CLI entry point for chrome with subcommand routing
package main

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
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
	_ "github.com/nathants/chrome/cmd/instances"
	_ "github.com/nathants/chrome/cmd/launch"
	_ "github.com/nathants/chrome/cmd/list"
	_ "github.com/nathants/chrome/cmd/navigate"
	_ "github.com/nathants/chrome/cmd/network"
	_ "github.com/nathants/chrome/cmd/newtab"
	_ "github.com/nathants/chrome/cmd/quit"
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
	fmt.Fprintln(os.Stderr, "  -p, --port PORT                          # Chrome debug port (default: 9222, env: CHROME_PORT)")
	fmt.Fprintln(os.Stderr, "  -t, --target URL_PREFIX                  # Select tab by URL prefix (env: CHROME_TARGET)")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Multi-Instance Usage:")
	fmt.Fprintln(os.Stderr, "  chrome launch --port 9223 --user-data-dir ~/.chrome-twitter")
	fmt.Fprintln(os.Stderr, "  chrome -p 9223 newtab https://x.com      # Use different port")
	fmt.Fprintln(os.Stderr, "  chrome instances                         # List running Chrome instances")
	fmt.Fprintln(os.Stderr, "  chrome -p 9223 quit                      # Quit Chrome on port 9223")
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
	port := ""
	for len(args) > 0 {
		arg := args[0]
		if arg == "-h" || arg == "--help" {
			usage()
			os.Exit(0)
		}
		if arg == "-p" || arg == "--port" {
			if len(args) < 2 {
				fmt.Fprintln(os.Stderr, "error: --port requires a value")
				os.Exit(1)
			}
			port = args[1]
			args = args[2:]
			continue
		}
		if strings.HasPrefix(arg, "--port=") {
			port = strings.TrimPrefix(arg, "--port=")
			args = args[1:]
			continue
		}
		if strings.HasPrefix(arg, "-p=") {
			port = strings.TrimPrefix(arg, "-p=")
			args = args[1:]
			continue
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
	if strings.TrimSpace(port) != "" {
		// Validate port is a number in valid range
		p, err := strconv.Atoi(port)
		if err != nil || p <= 0 || p >= 65536 {
			fmt.Fprintf(os.Stderr, "error: invalid port: %s (must be 1-65535)\n", port)
			os.Exit(1)
		}
		err = os.Setenv("CHROME_PORT", port)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
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
