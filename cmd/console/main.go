// console provides Chrome console log capture command.
package console

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/nathants/chrome/lib"
)

func init() {
	lib.Commands["console"] = console
	lib.Args["console"] = consoleArgs{}
}

type consoleArgs struct {
	Duration int  `arg:"-d,--duration" default:"5" help:"duration in seconds to capture logs"`
	Follow   bool `arg:"-f,--follow" help:"follow mode, capture logs continuously"`
}

func (consoleArgs) Description() string {
	return `console - Capture console logs

Captures console.log, console.warn, console.error and exceptions from the page.

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

	messages := make(chan ConsoleMessage, 100)

	chromedp.ListenTarget(ctx, func(ev interface{}) {
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
						_ = json.Unmarshal(arg.Value, &val)
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
		}
	})

	if err := chromedp.Run(ctx, runtime.Enable()); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if args.Follow {
		for {
			msg := <-messages
			output, _ := json.MarshalIndent(msg, "", "  ")
			fmt.Println(string(output))
		}
	}

	deadline := time.After(time.Duration(args.Duration) * time.Second)
	for {
		select {
		case msg := <-messages:
			output, _ := json.MarshalIndent(msg, "", "  ")
			fmt.Println(string(output))
		case <-deadline:
			return
		}
	}
}
