// lib provides common utilities and command registration for chrome cli
// commands register themselves in init() using Commands and Args maps
//
// ARCHITECTURE:
// - Each command is a separate package in cmd/ directory
// - Commands register themselves via init() functions
// - All commands share common context setup and target resolution
// - Supports both remote Chrome (port 9222) and headless mode
//
// TARGET RESOLUTION:
// - Empty selector: Uses first available page tab (non chrome://)
// - URL prefix: Selects first tab whose URL starts with the given prefix (case-insensitive)
// - CHROME_TARGET env var: Used when -t flag is empty
//
// CONTEXT LIFECYCLE:
// - Remote mode: Creates RemoteAllocator, attaches to existing Chrome
// - Headless mode: Creates ExecAllocator, launches new Chrome process
// - Tab contexts are created per-command and cancelled after execution
// - Cancelling tab context does NOT close the tab (tabs persist)
//
// CRITICAL LEARNINGS:
// - Chrome tabs persist between commands when using remote debugging
// - filterPageTargets removes chrome:// and chrome-untrusted:// URLs
// - Target selection prefers "attached" tabs from chromedp.Targets()
package lib

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

const (
	DefaultTimeout = 30 * time.Second
	ChromeURL      = "http://localhost:9222"
)

var cdpHTTPClient = &http.Client{Timeout: 5 * time.Second}

type ArgsStruct interface {
	Description() string
}

var Commands = make(map[string]func())
var Args = map[string]ArgsStruct{}

type ChromeTarget struct {
	ID                   string `json:"id"`
	Type                 string `json:"type"`
	URL                  string `json:"url"`
	Title                string `json:"title"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

// TargetArgs provides -t/--target flag for tab selection
// Embed this in command arg structs to enable tab targeting
// Example: type myArgs struct { lib.TargetArgs; MyField string }
type TargetArgs struct {
	Target string `arg:"-t,--target" help:"URL prefix to select tab (first match wins)"`
}

// Selector returns the trimmed target string for resolution
func (t TargetArgs) Selector() string {
	return strings.TrimSpace(t.Target)
}

func ResolveTargetWithArgs(args TargetArgs) (string, string, error) {
	return ResolveTarget(args.Selector(), nil)
}

// SetupContext creates a chromedp context with DefaultTimeout.
// Uses RemoteAllocator if Chrome is running on port 9222, otherwise ExecAllocator.
func SetupContext() (context.Context, context.CancelFunc) {
	return SetupContextWithTimeout(DefaultTimeout)
}

// SetupContextWithTimeout creates a chromedp context.
// If timeout <= 0, the context has no deadline (caller must cancel).
func SetupContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}

	if IsChromeRunning() {
		allocCtx, allocCancel := chromedp.NewRemoteAllocator(ctx, ChromeURL)

		targetID, _, _ := ResolveTarget("", nil)

		var browserCtx context.Context

		if targetID != "" {
			browserCtx, _ = chromedp.NewContext(allocCtx, chromedp.WithTargetID(target.ID(targetID)))
		} else {
			browserCtx, _ = chromedp.NewContext(allocCtx)
		}

		// Intentionally do not call the chromedp context cancel function here.
		// In remote-debugging mode we want tabs to persist between CLI invocations.
		combinedCancel := func() {
			allocCancel()
			cancel()
		}

		return browserCtx, combinedCancel
	}

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, opts...)
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)

	combinedCancel := func() {
		browserCancel()
		allocCancel()
		cancel()
	}

	return browserCtx, combinedCancel
}

func IsChromeRunning() bool {
	resp, err := cdpHTTPClient.Get(ChromeURL + "/json/version")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

func FetchTargets() ([]ChromeTarget, error) {
	resp, err := cdpHTTPClient.Get(ChromeURL + "/json/list")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var targets []ChromeTarget
	if err := json.Unmarshal(body, &targets); err != nil {
		return nil, err
	}

	return targets, nil
}

func FindFirstPageTarget() string {
	targets, err := FetchTargets()
	if err != nil {
		return ""
	}

	pages := filterPageTargets(targets)
	if len(pages) == 0 {
		return ""
	}

	return pages[0].ID
}

func FindPreferredTarget() string {
	id, _, err := ResolveTarget("", nil)
	if err != nil {
		return ""
	}
	return id
}

// ResolveTarget finds the target ID matching the selector
// Returns: (targetID, reason, error)
//
// Selection priority:
// 1. If selector provided: try to match by URL prefix
// 2. If CHROME_TARGET env var set: try to match by URL prefix
// 3. Check chromedp.Targets() for attached tabs (preferred)
// 4. Fall back to first page target
//
// Returns empty targetID with reason string if no match found
func ResolveTarget(selector string, env map[string]string) (string, string, error) {
	selected := strings.TrimSpace(selector)
	if selected == "" && env != nil {
		selected = strings.TrimSpace(env["CHROME_TARGET"])
	}
	if selected == "" {
		selected = strings.TrimSpace(os.Getenv("CHROME_TARGET"))
	}

	targets, err := FetchTargets()
	if err != nil {
		return "", "", err
	}

	pages := filterPageTargets(targets)
	if len(pages) == 0 {
		return "", "no page targets", nil
	}

	if selected != "" {
		if id := matchTargetBySelector(pages, selected); id != "" {
			return id, fmt.Sprintf("matched selector %q", selected), nil
		}

		// Better error message - show available tabs
		var availTabs []string
		for _, p := range pages {
			availTabs = append(availTabs, p.URL)
		}
		return "", fmt.Sprintf("no tab URL starts with %q. Available: %s", selected, strings.Join(availTabs, ", ")), nil
	}

	infos, err := fetchTargetInfos()
	if err == nil {
		if id := selectPreferredFromInfo(pages, infos); id != "" {
			return id, "preferred attached tab", nil
		}
	}

	return pages[0].ID, "first page tab", nil
}

// filterPageTargets returns only page-type targets, excluding chrome:// URLs
// This removes internal Chrome pages like settings, new tab, extensions, etc.
func filterPageTargets(targets []ChromeTarget) []ChromeTarget {
	var pages []ChromeTarget
	for _, t := range targets {
		if t.Type != "page" {
			continue
		}
		if strings.HasPrefix(t.URL, "chrome://") || strings.HasPrefix(t.URL, "chrome-untrusted://") {
			continue
		}
		pages = append(pages, t)
	}
	return pages
}

func matchTargetBySelector(pages []ChromeTarget, selector string) string {
	if selector == "" {
		return ""
	}

	// Match by URL prefix (case-insensitive), return first match
	selectorLower := strings.ToLower(selector)
	for _, t := range pages {
		if strings.HasPrefix(strings.ToLower(t.URL), selectorLower) {
			return t.ID
		}
	}

	return ""
}

func selectPreferredFromInfo(pages []ChromeTarget, infos []*target.Info) string {
	if len(pages) == 0 {
		return ""
	}

	pageSet := make(map[string]struct{}, len(pages))
	for _, p := range pages {
		pageSet[p.ID] = struct{}{}
	}

	for _, info := range infos {
		if info == nil || info.Type != "page" {
			continue
		}
		id := info.TargetID.String()
		if _, ok := pageSet[id]; !ok {
			continue
		}
		if info.Attached {
			return id
		}
	}

	for _, info := range infos {
		if info == nil || info.Type != "page" {
			continue
		}
		id := info.TargetID.String()
		if _, ok := pageSet[id]; ok {
			return id
		}
	}

	return ""
}

func fetchTargetInfos() ([]*target.Info, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	allocCtx, allocCancel := chromedp.NewRemoteAllocator(ctx, ChromeURL)
	defer allocCancel()

	ctx2, cancel2 := chromedp.NewContext(allocCtx)
	defer cancel2()

	infos, err := chromedp.Targets(ctx2)
	if err != nil {
		return nil, err
	}

	return infos, nil
}

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[:8]
}

func ListTabs() error {
	if !IsChromeRunning() {
		return fmt.Errorf("Chrome not running on port 9222")
	}

	targets, err := FetchTargets()
	if err != nil {
		return err
	}

	pages := filterPageTargets(targets)
	if len(pages) == 0 {
		fmt.Println("no page tabs")
		return nil
	}

	infos, err := fetchTargetInfos()
	if err != nil {
		infos = nil
	}

	infoByID := map[string]*target.Info{}
	for _, info := range infos {
		if info == nil {
			continue
		}
		infoByID[info.TargetID.String()] = info
	}

	preferred, _, _ := ResolveTarget("", nil)

	for _, page := range pages {
		marker := " "
		if preferred != "" && page.ID == preferred {
			marker = "*"
		}

		title := page.Title
		if title == "" {
			title = "(no title)"
		}

		fmt.Printf("%s[%s] %s\n  %s\n", marker, shortID(page.ID), title, page.URL)

		if info := infoByID[page.ID]; info != nil {
			status := "detached"
			if info.Attached {
				status = "attached"
			}
			fmt.Printf("  status: %s\n", status)
		}
	}

	return nil
}

// EnsureTargetContext creates a new context targeting a specific tab
// If selector is empty, returns the provided context unchanged
// Otherwise resolves the selector and creates a new context with WithTargetID
//
// IMPORTANT: The returned cancel function is a NO-OP for remote Chrome.
// This ensures tabs persist between commands. Cancelling the context would
// close the tab, which breaks multi-step automation workflows.
func EnsureTargetContext(ctx context.Context, selector string) (context.Context, func(), error) {
	sel := strings.TrimSpace(selector)
	if sel == "" {
		return ctx, func() {}, nil
	}

	if !IsChromeRunning() {
		return nil, nil, fmt.Errorf("target selection requires Chrome on %s", ChromeURL)
	}

	id, reason, err := ResolveTarget(sel, nil)
	if err != nil {
		return nil, nil, err
	}
	if id == "" {
		return nil, nil, errors.New(reason)
	}

	// Create context targeting specific tab
	// DO NOT cancel this context - we want tabs to persist for remote Chrome
	if existing := chromedp.FromContext(ctx); existing != nil {
		if existing.Target != nil && existing.Target.TargetID.String() == id {
			return ctx, func() {}, nil
		}
	}

	tabCtx, _ := chromedp.NewContext(ctx, chromedp.WithTargetID(target.ID(id)))
	return tabCtx, func() {}, nil
}
