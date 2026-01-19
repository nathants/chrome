// network provides Chrome network monitoring command.
package network

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/nathants/chrome/lib"
)

func init() {
	lib.Commands["network"] = networkCmd
	lib.Args["network"] = networkArgs{}
}

type networkArgs struct {
	lib.TargetArgs
	Duration int  `arg:"-d,--duration" default:"5" help:"duration in seconds to monitor"`
	Follow   bool `arg:"-f,--follow" help:"follow mode, monitor continuously"`
	Eval     string `arg:"--eval" help:"JavaScript to evaluate after enabling network capture"`
}

func (networkArgs) Description() string {
	return `network - Monitor network requests

Captures HTTP requests and responses from the page.
Output is JSON, one object per line (NDJSON).
Use --eval to run JavaScript after capture starts (handy for triggering requests).

Example:
  chrome network                    # Monitor for 5 seconds
  chrome network -d 10              # Monitor for 10 seconds
  chrome network -f                 # Follow mode (continuous, Ctrl+C to stop)`
}

type NetworkEvent struct {
	Type       string    `json:"type"`
	RequestID  string    `json:"requestId"`
	URL        string    `json:"url,omitempty"`
	Method     string    `json:"method,omitempty"`
	Status     int64     `json:"status,omitempty"`
	StatusText string    `json:"statusText,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

func networkCmd() {
	var args networkArgs
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

	events := make(chan NetworkEvent, 100)

	chromedp.ListenTarget(targetCtx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *network.EventRequestWillBeSent:
			evt := NetworkEvent{
				Type:      "request",
				RequestID: string(ev.RequestID),
				URL:       ev.Request.URL,
				Method:    ev.Request.Method,
				Timestamp: time.Now(),
			}
			select {
			case events <- evt:
			default:
			}
		case *network.EventResponseReceived:
			evt := NetworkEvent{
				Type:       "response",
				RequestID:  string(ev.RequestID),
				URL:        ev.Response.URL,
				Status:     ev.Response.Status,
				StatusText: ev.Response.StatusText,
				Timestamp:  time.Now(),
			}
			select {
			case events <- evt:
			default:
			}
		case *network.EventLoadingFailed:
			evt := NetworkEvent{
				Type:      "failed",
				RequestID: string(ev.RequestID),
				Timestamp: time.Now(),
			}
			select {
			case events <- evt:
			default:
			}
		}
	})

	if err := chromedp.Run(targetCtx, network.Enable()); err != nil {
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
			evt := <-events
			lib.PrintJSONLine(evt)
		}
	}

	deadline := time.After(time.Duration(args.Duration) * time.Second)
	for {
		select {
		case evt := <-events:
			lib.PrintJSONLine(evt)
		case <-deadline:
			return
		}
	}
}
