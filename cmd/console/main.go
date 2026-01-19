// console provides Chrome console log capture command.
package console

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	cdplog "github.com/chromedp/cdproto/log"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/nathants/chrome/lib"
)

func init() {
	lib.Commands["console"] = console
	lib.Args["console"] = consoleArgs{}
}

type consoleArgs struct {
	lib.TargetArgs
	Duration int  `arg:"-d,--duration" default:"5" help:"duration in seconds to capture logs"`
	Follow   bool `arg:"-f,--follow" help:"follow mode, capture logs continuously"`
	Eval     string `arg:"--eval" help:"JavaScript to evaluate after enabling log capture"`
}

func (consoleArgs) Description() string {
	return `console - Capture console logs

Captures console.log, console.warn, console.error, exceptions, and browser log
events (CSP violations, security errors, deprecation warnings, etc) from the page.
Output is JSON, one object per line (NDJSON).
Use --eval to run JavaScript after capture starts (handy for triggering logs).

Example:
  chrome console                    # Capture for 5 seconds
  chrome console -d 10              # Capture for 10 seconds
  chrome console -f                 # Follow mode (continuous, Ctrl+C to stop)`
}

type ConsoleMessage struct {
	Type      string      `json:"type"`
	Message   string      `json:"message,omitempty"`
	Args      interface{} `json:"args,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Level     string      `json:"level,omitempty"`
}

func console() {
	var args consoleArgs
	arg.MustParse(&args)

	ctxTimeout := lib.DefaultTimeout
	if args.Follow {
		ctxTimeout = 0
	} else {
		d := time.Duration(args.Duration)*time.Second + 5*time.Second
		if d > ctxTimeout {
			ctxTimeout = d
		}
	}

	ctx, cancel := lib.SetupContextWithTimeout(ctxTimeout)
	defer cancel()

	targetCtx, targetCancel, err := lib.EnsureTargetContext(ctx, args.TargetArgs.Selector())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer targetCancel()

	messages := make(chan ConsoleMessage, 100)

	chromedp.ListenTarget(targetCtx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *runtime.EventConsoleAPICalled:
			msg := ConsoleMessage{
				Type:      string(ev.Type),
				Timestamp: time.Now(),
			}

			// Extract argument values.
			if len(ev.Args) > 0 {
				var args []interface{}
				for _, arg := range ev.Args {
					var val interface{}
					if arg.Value != nil {
						err := json.Unmarshal(arg.Value, &val)
						if err != nil {
							val = string(arg.Value)
						}
						args = append(args, val)
					} else {
						args = append(args, arg.Type.String())
					}
				}
				if len(args) == 1 {
					if s, ok := args[0].(string); ok {
						msg.Message = s
					} else {
						msg.Args = args[0]
					}
				} else {
					msg.Args = args
				}
			}

			select {
			case messages <- msg:
			default:
			}
		case *runtime.EventExceptionThrown:
			msg := ConsoleMessage{
				Type:      "exception",
				Level:     "error",
				Timestamp: time.Now(),
			}
			if ev.ExceptionDetails.Exception != nil {
				msg.Message = ev.ExceptionDetails.Exception.Description
			} else {
				msg.Message = ev.ExceptionDetails.Text
			}

			select {
			case messages <- msg:
			default:
			}
		case *cdplog.EventEntryAdded:
			// Capture Log domain events (CSP violations, security errors, etc).
			msg := ConsoleMessage{
				Type:      string(ev.Entry.Source),
				Level:     string(ev.Entry.Level),
				Message:   ev.Entry.Text,
				Timestamp: time.Now(),
			}

			select {
			case messages <- msg:
			default:
			}
		}
	})

	if err := chromedp.Run(targetCtx, runtime.Enable(), cdplog.Enable()); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if strings.TrimSpace(args.Eval) != "" {
		err := chromedp.Run(targetCtx, chromedp.Evaluate(args.Eval, nil))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}

	if args.Follow {
		for {
			msg := <-messages
			lib.PrintJSONLine(msg)
		}
	}

	deadline := time.After(time.Duration(args.Duration) * time.Second)
	for {
		select {
		case msg := <-messages:
			lib.PrintJSONLine(msg)
		case <-deadline:
			return
		}
	}
}
