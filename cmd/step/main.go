// step runs a chrome action and captures an immediate screenshot.
package step

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"time"

	"github.com/nathants/chrome/lib"
)

func init() {
	lib.Commands["step"] = step
	lib.Args["step"] = stepArgs{}
}

type stepArgs struct {
	lib.TargetArgs
	OutputDir string `arg:"-o,--output-dir" help:"directory to store screenshots (default: ~/chrome-shots)"`
	Label     string `arg:"-l,--label" help:"label embedded in filename (default: action name)"`
	Note      string `arg:"-n,--note" help:"note stored with metadata"`
	Action    string `arg:"positional,required" help:"chrome command to execute (e.g. click, type, waitfor)"`
}

func (stepArgs) Description() string {
	return `step - Run an action and capture the result

Examples:
  chrome step navigate https://localhost:3000
  chrome step -t http://localhost:3000 click "button.submit"
  chrome step type "#name" "Alice"`
}

type parsedStep struct {
	target     string
	outputDir  string
	label      string
	note       string
	action     string
	actionArgs []string
}

func step() {
	parsed, err := parseStep(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	action := parsed.action
	target := parsed.target
	supportsTarget := commandSupportsTarget(action)
	actionArgs := append([]string{}, parsed.actionArgs...)
	if supportsTarget {
		actionArgs = applyTarget(actionArgs, target)
	}
	actionSummary := strings.TrimSpace(strings.Join(actionArgs, " "))
	if actionSummary != "" {
		fmt.Printf("action: chrome %s %s\n", action, actionSummary)
	} else {
		fmt.Printf("action: chrome %s\n", action)
	}

	if err := runSubcommand(action, actionArgs); err != nil {
		fmt.Fprintf(os.Stderr, "error executing action: %v\n", err)
		os.Exit(1)
	}

	label := parsed.label
	if label == "" {
		label = action
	}

	path, err := lib.PrepareScreenshotPath("", parsed.outputDir, label)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error preparing screenshot path: %v\n", err)
		os.Exit(1)
	}

	shotArgs := []string{"screenshot", "--path", path}
	if parsed.label != "" {
		shotArgs = append(shotArgs, "--label", parsed.label)
	}
	if parsed.note != "" {
		shotArgs = append(shotArgs, "--note", parsed.note)
	}
	if target != "" {
		shotArgs = append(shotArgs, "--target", target)
	}
	if err := runSubcommand(shotArgs[0], shotArgs[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error capturing screenshot: %v\n", err)
		os.Exit(1)
	}

	record := lib.StepRecord{
		Action:     action,
		Args:       append([]string{}, actionArgs...),
		Target:     target,
		Label:      label,
		Note:       parsed.note,
		Screenshot: path,
		CreatedAt:  time.Now().UTC(),
	}

	if err := lib.RememberStep(record); err != nil {
		fmt.Fprintf(os.Stderr, "warning: unable to persist metadata: %v\n", err)
	}

	fmt.Println(lib.StepSummary(record))
	fmt.Printf("metadata: %s\n", record.MetadataPath())
	if record.Note != "" {
		fmt.Printf("note: %s\n", record.Note)
	}
}

func parseStep(args []string) (parsedStep, error) {
	var parsed parsedStep
	pos := 0
	for pos < len(args) {
		tok := args[pos]
		if tok == "--" {
			pos++
			break
		}
		if !strings.HasPrefix(tok, "-") {
			break
		}
		if strings.HasPrefix(tok, "--target=") {
			value := strings.TrimPrefix(tok, "--target=")
			if value == "" {
				return parsedStep{}, errors.New("--target requires a value")
			}
			parsed.target = value
			pos++
			continue
		}
		if strings.HasPrefix(tok, "--output-dir=") {
			value := strings.TrimPrefix(tok, "--output-dir=")
			if value == "" {
				return parsedStep{}, errors.New("--output-dir requires a value")
			}
			parsed.outputDir = value
			pos++
			continue
		}
		if strings.HasPrefix(tok, "--label=") {
			value := strings.TrimPrefix(tok, "--label=")
			if value == "" {
				return parsedStep{}, errors.New("--label requires a value")
			}
			parsed.label = value
			pos++
			continue
		}
		if strings.HasPrefix(tok, "--note=") {
			value := strings.TrimPrefix(tok, "--note=")
			if value == "" {
				return parsedStep{}, errors.New("--note requires a value")
			}
			parsed.note = value
			pos++
			continue
		}
		switch tok {
		case "-t", "--target":
			pos++
			if pos >= len(args) {
				return parsedStep{}, errors.New("--target requires a value")
			}
			parsed.target = args[pos]
		case "-o", "--output-dir":
			pos++
			if pos >= len(args) {
				return parsedStep{}, errors.New("--output-dir requires a value")
			}
			parsed.outputDir = args[pos]
		case "-l", "--label":
			pos++
			if pos >= len(args) {
				return parsedStep{}, errors.New("--label requires a value")
			}
			parsed.label = args[pos]
		case "-n", "--note":
			pos++
			if pos >= len(args) {
				return parsedStep{}, errors.New("--note requires a value")
			}
			parsed.note = args[pos]
		default:
			return parsedStep{}, fmt.Errorf("unknown step option %q", tok)
		}
		pos++
	}

	if pos >= len(args) {
		return parsedStep{}, errors.New("action is required")
	}

	parsed.action = args[pos]
	pos++

	if parsed.target == "" {
		parsed.target = strings.TrimSpace(os.Getenv("CHROME_TARGET"))
	}

	if pos < len(args) {
		parsed.actionArgs = append(parsed.actionArgs, args[pos:]...)
	}

	return parsed, nil
}

func runSubcommand(name string, args []string) error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(execPath, append([]string{name}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	return cmd.Run()
}

func applyTarget(args []string, target string) []string {
	if target == "" {
		return append([]string{}, args...)
	}

	for _, arg := range args {
		if arg == "-t" || arg == "--target" {
			return append([]string{}, args...)
		}
		if strings.HasPrefix(arg, "-t=") || strings.HasPrefix(arg, "--target=") {
			return append([]string{}, args...)
		}
	}

	updated := append([]string{}, args...)
	return append([]string{"-t", target}, updated...)
}

func commandSupportsTarget(name string) bool {
	argsStruct, ok := lib.Args[name]
	if !ok {
		return false
	}
	t := reflect.TypeOf(argsStruct)
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return false
	}
	_, found := t.FieldByName("TargetArgs")
	return found
}
