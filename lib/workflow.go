package lib

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/gorilla/websocket"
)

var labelCleanup = regexp.MustCompile("[^a-z0-9-]+")

// StepRecord captures the outcome of a chrome action + screenshot loop.
type StepRecord struct {
	Action     string    `json:"action"`
	Args       []string  `json:"args"`
	Target     string    `json:"target"`
	Label      string    `json:"label"`
	Note       string    `json:"note"`
	Screenshot string    `json:"screenshot"`
	CreatedAt  time.Time `json:"created_at"`
}

func (record StepRecord) MetadataPath() string {
	return record.Screenshot + ".json"
}

func CaptureScreenshot(selector string, path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return err
	}

	if IsChromeRunning() {
		if err := captureScreenshotRemote(selector, absPath); err == nil {
			return nil
		}
	}

	ctx, cancel := SetupContext()
	defer cancel()

	targetCtx, targetCancel, err := EnsureTargetContext(ctx, selector)
	if err != nil {
		return err
	}
	defer targetCancel()

	var buf []byte
	if err := chromedp.Run(targetCtx, chromedp.CaptureScreenshot(&buf)); err != nil {
		return err
	}

	return os.WriteFile(absPath, buf, 0644)
}

func captureScreenshotRemote(selector string, path string) error {
	targetID, reason, err := ResolveTarget(selector, nil)
	if err != nil {
		return err
	}
	if targetID == "" {
		return errors.New(reason)
	}

	targets, err := FetchTargets()
	if err != nil {
		return err
	}

	var wsURL string
	for _, t := range targets {
		if t.ID == targetID {
			wsURL = strings.TrimSpace(t.WebSocketDebuggerURL)
			break
		}
	}

	if wsURL == "" {
		return fmt.Errorf("target %s missing websocket debugger url", targetID)
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.WriteJSON(map[string]any{"id": 1, "method": "Page.enable"}); err != nil {
		return err
	}
	_ = conn.WriteJSON(map[string]any{"id": 2, "method": "Page.bringToFront"})
	if err := conn.WriteJSON(map[string]any{
		"id":     3,
		"method": "Page.captureScreenshot",
		"params": map[string]any{
			"format":      "png",
			"fromSurface": true,
		},
	}); err != nil {
		return err
	}

	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		var resp struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(data, &resp); err != nil {
			continue
		}
		if resp.ID != 3 {
			continue
		}
		if resp.Error != nil {
			return fmt.Errorf("capture screenshot error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		var payload struct {
			Data string `json:"data"`
		}
		if err := json.Unmarshal(resp.Result, &payload); err != nil {
			return err
		}
		if payload.Data == "" {
			return errors.New("empty screenshot data")
		}
		bytes, err := base64.StdEncoding.DecodeString(payload.Data)
		if err != nil {
			return err
		}
		return os.WriteFile(path, bytes, 0644)
	}
}

func DefaultShotsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "chrome-shots")
	}
	return filepath.Join(home, "chrome-shots")
}

func PrepareShotsDir(dir string) (string, error) {
	targetDir := strings.TrimSpace(dir)
	if targetDir == "" {
		targetDir = DefaultShotsDir()
	}
	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return "", err
	}
	err = os.MkdirAll(absDir, 0755)
	if err != nil {
		return "", err
	}
	return absDir, nil
}

func PrepareScreenshotPath(path string, dir string, label string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed != "" {
		absPath, err := filepath.Abs(trimmed)
		if err != nil {
			return "", err
		}
		err = os.MkdirAll(filepath.Dir(absPath), 0755)
		if err != nil {
			return "", err
		}
		return absPath, nil
	}

	shotsDir, err := PrepareShotsDir(dir)
	if err != nil {
		return "", err
	}

	sanitized := sanitizeLabel(label)
	if sanitized == "" {
		sanitized = "shot"
	}
	timestamp := time.Now().UTC().Format("20060102-150405.000")
	timestamp = strings.ReplaceAll(timestamp, ".", "_")
	filename := fmt.Sprintf("%s-%s.png", timestamp, sanitized)
	return filepath.Join(shotsDir, filename), nil
}

func sanitizeLabel(label string) string {
	lower := strings.ToLower(strings.TrimSpace(label))
	if lower == "" {
		return ""
	}
	replaced := labelCleanup.ReplaceAllString(lower, "-")
	compact := strings.Trim(replaced, "-")
	compact = strings.ReplaceAll(compact, "--", "-")
	return compact
}

func SaveStepRecord(record StepRecord) error {
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	record.CreatedAt = record.CreatedAt.UTC()
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(record.MetadataPath(), data, 0644)
}

func SaveLastStep(record StepRecord) error {
	cache, err := cacheDir()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(cache, "last-step.json")
	return os.WriteFile(path, data, 0644)
}

func LoadLastStep() (StepRecord, error) {
	cache, err := cacheDir()
	if err != nil {
		return StepRecord{}, err
	}
	path := filepath.Join(cache, "last-step.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return StepRecord{}, err
	}
	var record StepRecord
	err = json.Unmarshal(data, &record)
	if err != nil {
		return StepRecord{}, err
	}
	return record, nil
}

func LoadStepMetadata(path string) (StepRecord, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return StepRecord{}, err
	}
	metaPath := absPath + ".json"
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return StepRecord{Screenshot: absPath}, nil
		}
		return StepRecord{}, err
	}
	var record StepRecord
	err = json.Unmarshal(data, &record)
	if err != nil {
		return StepRecord{}, err
	}
	if record.Screenshot == "" {
		record.Screenshot = absPath
	}
	return record, nil
}

func RememberStep(record StepRecord) error {
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	err := SaveStepRecord(record)
	if err != nil {
		return err
	}
	return SaveLastStep(record)
}

// cacheDir returns the cache directory ($XDG_CACHE_HOME/chrome-cli or ~/.cache/chrome-cli).
func cacheDir() (string, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return "", err
		}
		cache = filepath.Join(home, ".cache")
	}
	cache = filepath.Join(cache, "chrome-cli")
	err = os.MkdirAll(cache, 0755)
	if err != nil {
		return "", err
	}
	return cache, nil
}

func StepSummary(record StepRecord) string {
	rel := record.Screenshot
	cwd, err := os.Getwd()
	if err == nil {
		if relPath, relErr := filepath.Rel(cwd, record.Screenshot); relErr == nil {
			rel = relPath
		}
	}
	argsText := strings.TrimSpace(strings.Join(record.Args, " "))
	timestamp := "unknown"
	if !record.CreatedAt.IsZero() {
		timestamp = record.CreatedAt.UTC().Format(time.RFC3339)
	}
	if argsText != "" {
		return fmt.Sprintf("[%s] %s %s -> %s", timestamp, record.Action, argsText, rel)
	}
	return fmt.Sprintf("[%s] %s -> %s", timestamp, record.Action, rel)
}
