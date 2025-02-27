package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
)

type FFProbeOutput struct {
	Streams []struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"streams"`
}

func getVideoAspectRatio(filePath string) (string, error) {
	cmd := exec.Command("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", filePath)

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffprobe command failed: %w", err)
	}

	var probe FFProbeOutput
	if err := json.Unmarshal(out.Bytes(), &probe); err != nil {
		return "", fmt.Errorf("failed to unmarshal ffprobe output: %w", err)
	}

	for _, stream := range probe.Streams {
		if stream.Width > 0 && stream.Height > 0 {
			ratio := float64(stream.Width) / float64(stream.Height)
			const tolerance = 0.1

			if math.Abs(ratio-(16.0/9.0)) < tolerance {
				return "16:9", nil
			} else if math.Abs(ratio-(9.0/16.0)) < tolerance {
				return "9:16", nil
			} else {
				return "other", nil
			}
		}
	}

	return "", fmt.Errorf("no valid video stream found")
}

func processVideoForFastStart(filePath string) (string, error) {
	outputPath := filePath + ".processing"

	cmd := exec.Command("ffmpeg",
		"-i", filePath,
		"-c", "copy",
		"-movflags", "faststart",
		"-f", "mp4",
		outputPath,
	)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg command failed: %w", err)
	}

	return outputPath, nil
}
