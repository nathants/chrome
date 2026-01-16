package lib

import (
	"bufio"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	slideshowFrameDurationSeconds = 5
	subtitleFontName              = "DejaVu Sans"
	subtitleFontSize              = 32
)

// LoadStepRecordsFromDir loads all StepRecord metadata from a directory.
func LoadStepRecordsFromDir(dir string) ([]StepRecord, error) {
	trimmed := strings.TrimSpace(dir)
	if trimmed == "" {
		return nil, errors.New("input directory is required")
	}
	absDir, err := filepath.Abs(trimmed)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return nil, err
	}

	var records []StepRecord
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		metaPath := filepath.Join(absDir, entry.Name())
		record, loadErr := LoadStepMetadata(strings.TrimSuffix(metaPath, ".json"))
		if loadErr != nil {
			continue
		}
		if record.Screenshot == "" {
			continue
		}
		if _, statErr := os.Stat(record.Screenshot); statErr != nil {
			continue
		}
		records = append(records, record)
	}

	sort.SliceStable(records, func(i, j int) bool {
		a := records[i].CreatedAt
		b := records[j].CreatedAt
		switch {
		case a.IsZero() && b.IsZero():
			return records[i].Screenshot < records[j].Screenshot
		case a.IsZero():
			return false
		case b.IsZero():
			return true
		default:
			if a.Equal(b) {
				return records[i].Screenshot < records[j].Screenshot
			}
			return a.Before(b)
		}
	})

	return records, nil
}

// GenerateSlideshow renders an mp4 slideshow from the provided step records at the requested fps.
func GenerateSlideshow(records []StepRecord, outputPath string, fps int) error {
	if len(records) == 0 {
		return errors.New("no step records provided for slideshow")
	}
	if fps <= 0 {
		fps = 30
	}
	absOutput, err := filepath.Abs(strings.TrimSpace(outputPath))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(absOutput), 0755); err != nil {
		return err
	}

	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return errors.New("ffmpeg not found in PATH")
	}

	tempDir, err := os.MkdirTemp("", "chrome-slideshow-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	concatPath := filepath.Join(tempDir, "inputs.txt")
	captionsPath := filepath.Join(tempDir, "captions.srt")

	frameDuration := time.Duration(slideshowFrameDurationSeconds) * time.Second

	maxWidth, maxHeight, err := maxScreenshotDimensions(records)
	if err != nil {
		return err
	}
	if maxWidth%2 != 0 {
		maxWidth++
	}
	if maxHeight%2 != 0 {
		maxHeight++
	}

	if err := writeConcatFile(concatPath, records, frameDuration); err != nil {
		return err
	}
	if err := writeCaptionsFile(captionsPath, records, frameDuration); err != nil {
		return err
	}

	filterParts := []string{}
	if maxWidth > 0 && maxHeight > 0 {
		filterParts = append(filterParts, fmt.Sprintf("pad=%d:%d:(%d-iw)/2:(%d-ih)/2", maxWidth, maxHeight, maxWidth, maxHeight))
	}

	filterParts = append(filterParts, fmt.Sprintf("subtitles='%s':force_style='FontName=%s,FontSize=%d,PrimaryColour=\u0026H00FFFFFF\u0026,OutlineColour=\u0026H00000000\u0026,BorderStyle=3,Outline=1,Shadow=0,Alignment=2'",
		escapeForFilter(captionsPath), subtitleFontName, subtitleFontSize))
	filter := strings.Join(filterParts, ",")

	cmd := exec.Command(
		ffmpegPath,
		"-y",
		"-f", "concat",
		"-safe", "0",
		"-i", concatPath,
		"-r", fmt.Sprintf("%d", fps),
		"-vf", filter,
		"-pix_fmt", "yuv420p",
		"-c:v", "libx264",
		"-vsync", "cfr",
		absOutput,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func writeConcatFile(path string, records []StepRecord, frameDuration time.Duration) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	writer := bufio.NewWriter(file)
	for idx, record := range records {
		abs := record.Screenshot
		fmt.Fprintf(writer, "file '%s'\n", escapeForConcat(abs))
		fmt.Fprintf(writer, "duration %.3f\n", frameDuration.Seconds())
		if idx == len(records)-1 {
			fmt.Fprintf(writer, "file '%s'\n", escapeForConcat(abs))
		}
	}
	return writer.Flush()
}

func writeCaptionsFile(path string, records []StepRecord, frameDuration time.Duration) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	writer := bufio.NewWriter(file)
	for idx, record := range records {
		start := frameDuration * time.Duration(idx)
		end := frameDuration * time.Duration(idx+1)
		text := slideshowCaption(record)
		if text == "" {
			continue
		}
		fmt.Fprintf(writer, "%d\n", idx+1)
		fmt.Fprintf(writer, "%s --> %s\n", formatSRTTime(start), formatSRTTime(end))
		for _, line := range wrapText(text, 72) {
			fmt.Fprintln(writer, line)
		}
		fmt.Fprintln(writer)
	}
	return writer.Flush()
}

func maxScreenshotDimensions(records []StepRecord) (int, int, error) {
	maxWidth := 0
	maxHeight := 0

	for _, record := range records {
		file, err := os.Open(record.Screenshot)
		if err != nil {
			return 0, 0, fmt.Errorf("opening screenshot %s: %w", record.Screenshot, err)
		}
		cfg, _, err := image.DecodeConfig(file)
		_ = file.Close()
		if err != nil {
			return 0, 0, fmt.Errorf("decoding screenshot %s: %w", record.Screenshot, err)
		}
		if cfg.Width > maxWidth {
			maxWidth = cfg.Width
		}
		if cfg.Height > maxHeight {
			maxHeight = cfg.Height
		}
	}

	return maxWidth, maxHeight, nil
}

func escapeForConcat(path string) string {
	replaced := strings.ReplaceAll(path, "'", "'\\''")
	return replaced
}

func escapeForFilter(path string) string {
	replaced := strings.ReplaceAll(path, "'", "\\'\\''")
	replaced = strings.ReplaceAll(replaced, ":", "\\:")
	return replaced
}

func formatSRTTime(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	hours := int(d / time.Hour)
	d -= time.Duration(hours) * time.Hour
	minutes := int(d / time.Minute)
	d -= time.Duration(minutes) * time.Minute
	seconds := int(d / time.Second)
	d -= time.Duration(seconds) * time.Second
	millis := int(d / time.Millisecond)
	return fmt.Sprintf("%02d:%02d:%02d,%03d", hours, minutes, seconds, millis)
}

func slideshowCaption(record StepRecord) string {
	note := strings.TrimSpace(record.Note)
	if note != "" {
		return note
	}
	if record.Label != "" {
		return fmt.Sprintf("%s (%s)", record.Action, record.Label)
	}
	args := strings.TrimSpace(strings.Join(record.Args, " "))
	if args != "" {
		return fmt.Sprintf("%s %s", record.Action, args)
	}
	return strings.TrimSpace(record.Action)
}

func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	var current strings.Builder
	for _, word := range words {
		if current.Len() == 0 {
			current.WriteString(word)
			continue
		}
		if current.Len()+1+len(word) <= width {
			current.WriteByte(' ')
			current.WriteString(word)
			continue
		}
		lines = append(lines, current.String())
		current.Reset()
		current.WriteString(word)
	}
	if current.Len() > 0 {
		lines = append(lines, current.String())
	}
	return lines
}
