package common

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

func URLDotExt(u string) string {
	info, err := url.Parse(u)
	if err != nil {
		return ""
	}
	return filepath.Ext(info.Path)
}

var reWrongFileChars = regexp.MustCompile(`[\x{1}-\x{6}\x{e}-\x{19}\x{1b}-\x{1f}"<>\|\a\t\n\v\f\r\:\*\?\\\/]`)

func ReplaceWrongFileChars(stem string) string {
	stem = strings.ReplaceAll(strings.ReplaceAll(stem, "\\", "_"), "/", "_")
	return reWrongFileChars.ReplaceAllString(stem, "_")
}

func IsCtxDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

func MergeAV(ctx context.Context, v, a, output string) error {
	// ffmpeg -i input.mp4 -i input.mp3 -c copy output.mp4
	cmd := exec.CommandContext(ctx, LocalExecutableFile("ffmpeg"), "-i", v, "-i", a, "-c", "copy", output)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func ConvertToExt(ctx context.Context, input, ext string) string {
	if strings.ToLower(ext) == "audio" {
		ext = "mp3"
	}
	ext = strings.TrimPrefix(ext, ".")

	before, _, _ := strings.Cut(input, ".")
	output := filepath.Join(filepath.Dir(input), before+"."+ext)
	cmd := exec.CommandContext(ctx, LocalExecutableFile("ffmpeg"), "-i", input, output)
	if err := cmd.Run(); err != nil {
		fmt.Println(err)
		return input
	}
	os.Remove(input)
	return output
}

func ExecutableFile(name string) string {
	if strings.EqualFold(runtime.GOOS, "windows") {
		if filepath.Ext(name) != ".exe" {
			name += ".exe"
		}
		return name
	}
	return name
}

func LocalExecutableFile(name string) string {
	executablePath, err := os.Executable()
	if err != nil {
		return ""
	}
	executableDir := filepath.Dir(executablePath)
	return ExecutableFile(filepath.Join(executableDir, name))
}

func IsExistsFile(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
