package orchestrator

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

type process struct {
	cmd  *exec.Cmd
	mu   sync.Mutex
	done chan struct{}
}

func startProcess(command, cwd, logFile string, port int) (*process, error) {
	lf, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = cwd
	// Inherit the current environment and inject PORT so frameworks pick it up.
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PORT=%d", port),
		fmt.Sprintf("VITE_PORT=%d", port),
	)
	cmd.Stdout = io.MultiWriter(lf, os.Stdout)
	cmd.Stderr = io.MultiWriter(lf, os.Stderr)

	if err := cmd.Start(); err != nil {
		lf.Close()
		return nil, err
	}

	p := &process{cmd: cmd, done: make(chan struct{})}
	go func() {
		defer lf.Close()
		cmd.Wait()
		close(p.done)
	}()

	return p, nil
}

func (p *process) stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cmd.Process != nil {
		return p.cmd.Process.Kill()
	}
	return nil
}

func (p *process) running() bool {
	select {
	case <-p.done:
		return false
	default:
		return true
	}
}

func (p *process) wait() <-chan struct{} {
	return p.done
}
