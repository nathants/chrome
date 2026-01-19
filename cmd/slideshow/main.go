package slideshow

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/nathants/chrome/lib"
)

func init() {
	lib.Commands["slideshow"] = run
	lib.Args["slideshow"] = args{}
}

type args struct {
	ShotsDir string `arg:"-d,--shots-dir" help:"directory containing screenshots and metadata (default: ~/chrome-shots)"`
	Output   string `arg:"-o,--output" help:"output mp4 path (default: <shots-dir>/slideshow-<timestamp>.mp4)"`
	FPS      int    `arg:"-f,--fps" help:"frames per second for output video (default: 30)"`
	Verbose  bool   `arg:"--verbose" help:"show ffmpeg banner and progress output"`
}

func (args) Description() string {
	return `slideshow - build mp4 slideshow from captured steps

Examples:
  chrome slideshow
  chrome slideshow --shots-dir /tmp/run
  chrome slideshow --output /tmp/slideshow.mp4
  chrome slideshow --fps 30
  chrome slideshow --verbose`
}

func run() {
	var parsed args
	arg.MustParse(&parsed)

	dir := strings.TrimSpace(parsed.ShotsDir)
	if dir == "" {
		dir = lib.DefaultShotsDir()
	}

	records, err := lib.LoadStepRecordsFromDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading step records: %v\n", err)
		os.Exit(1)
	}
	if len(records) == 0 {
		fmt.Fprintf(os.Stderr, "error: no screenshots found in %s\n", dir)
		os.Exit(1)
	}

	output := strings.TrimSpace(parsed.Output)
	if output == "" {
		output = filepath.Join(dir, fmt.Sprintf("slideshow-%s.mp4", time.Now().UTC().Format("20060102-150405")))
	}

	fps := parsed.FPS
	if fps <= 0 {
		fps = 30
	}

	err = lib.GenerateSlideshow(records, output, fps, parsed.Verbose)
	if err != nil {
		var pathErr *os.PathError
		if errors.As(err, &pathErr) {
			fmt.Fprintf(os.Stderr, "filesystem error: %v\n", pathErr)
		} else {
			fmt.Fprintf(os.Stderr, "error generating slideshow: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Printf("slideshow created: %s\n", output)
}
