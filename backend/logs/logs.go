package logs

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func LogFile(dataDir, projectID string) string {
	return filepath.Join(dataDir, "logs", projectID+".log")
}

func ReadAll(dataDir, projectID string) (string, error) {
	b, err := os.ReadFile(LogFile(dataDir, projectID))
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

func ReadTail(dataDir, projectID string, n int) ([]string, error) {
	f, err := os.Open(LogFile(dataDir, projectID))
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if n > 0 && n < len(lines) {
		lines = lines[len(lines)-n:]
	}
	return lines, nil
}

func Clear(dataDir, projectID string) error {
	path := LogFile(dataDir, projectID)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	return os.WriteFile(path, []byte{}, 0644)
}

// StreamPath returns the log file path for use by callers that want to tail it.
func StreamPath(dataDir, projectID string) string {
	return fmt.Sprintf("%s/logs/%s.log", dataDir, projectID)
}
