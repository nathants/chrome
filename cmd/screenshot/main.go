// screenshot captures Chrome screenshots with metadata and safe defaults.
package screenshot

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/nathants/chrome/lib"
)

func init() {
	lib.Commands["screenshot"] = screenshot
	lib.Args["screenshot"] = screenshotArgs{}
}

type screenshotArgs struct {
	lib.TargetArgs
	Path      string `arg:"--path" help:"exact file path for screenshot (overrides output dir)"`
	OutputDir string `arg:"-o,--output-dir" help:"directory to store screenshots (default: ~/chrome-shots)"`
	Label     string `arg:"-l,--label" help:"label embedded in filename"`
	Note      string `arg:"-n,--note" help:"note saved in metadata"`
}

func (screenshotArgs) Description() string {
	return `screenshot - Capture a screenshot with metadata

Examples:
  chrome screenshot                                  # writes to ~/chrome-shots/<timestamp>-shot.png
  chrome screenshot --label after-login             # include label in metadata
  chrome screenshot --path /tmp/latest.png           # explicit path
  chrome screenshot -t http://localhost --note "after submit"        # annotate metadata`
}

func screenshot() {
	var args screenshotArgs
	arg.MustParse(&args)

	path, err := lib.PrepareScreenshotPath(args.Path, args.OutputDir, effectiveLabel(args.Label))
	if err != nil {
		fmt.Fprintf(os.Stderr, "error preparing screenshot path: %v\n", err)
		os.Exit(1)
	}

	err = lib.CaptureScreenshot(args.TargetArgs.Selector(), path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error capturing screenshot: %v\n", err)
		os.Exit(1)
	}

	record := lib.StepRecord{
		Action:     "screenshot",
		Args:       buildScreenshotArgs(args),
		Target:     args.TargetArgs.Selector(),
		Label:      effectiveLabel(args.Label),
		Note:       args.Note,
		Screenshot: path,
		CreatedAt:  time.Now().UTC(),
	}

	err = lib.RememberStep(record)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: unable to persist metadata: %v\n", err)
	}

	fmt.Printf("saved %s\n", record.Screenshot)
	if record.Note != "" {
		fmt.Printf("note: %s\n", record.Note)
	}
}

func effectiveLabel(label string) string {
	trimmed := strings.TrimSpace(label)
	if trimmed == "" {
		return "shot"
	}
	return trimmed
}

func buildScreenshotArgs(args screenshotArgs) []string {
	var collected []string

	if args.Path != "" {
		collected = append(collected, fmt.Sprintf("--path=%s", args.Path))
	}
	if args.OutputDir != "" {
		collected = append(collected, fmt.Sprintf("--output-dir=%s", args.OutputDir))
	}
	if args.Label != "" {
		collected = append(collected, fmt.Sprintf("--label=%s", args.Label))
	}
	if args.Note != "" {
		collected = append(collected, fmt.Sprintf("--note=%s", args.Note))
	}
	target := args.TargetArgs.Selector()
	if target != "" {
		collected = append(collected, fmt.Sprintf("--target=%s", target))
	}

	return collected
}
