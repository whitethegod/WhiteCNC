package utils

import (
	"bufio"
	"os"
)

var spinnerFrames []string
var spinnerIndex int

func LoadSpinner(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var frames []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		frames = append(frames, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	spinnerFrames = frames
	spinnerIndex = 0
	return nil
}

func GetNextSpinnerFrame() string {
	if len(spinnerFrames) == 0 {
		return "" 
	}
	frame := spinnerFrames[spinnerIndex]
	spinnerIndex = (spinnerIndex + 1) % len(spinnerFrames) 
	return frame
}
