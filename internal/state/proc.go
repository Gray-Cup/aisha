package state

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"

	"aisha/internal/config"
)

// ─────────────────────────────────────────────
//  Process manager
// ─────────────────────────────────────────────

const maxOutputLines = 500

type managedProc struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	output []string
	done   bool
}

func (mp *managedProc) appendLine(line string) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.output = append(mp.output, line)
	if len(mp.output) > maxOutputLines {
		mp.output = mp.output[len(mp.output)-maxOutputLines:]
	}
}

func (mp *managedProc) getOutput() []string {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	cp := make([]string, len(mp.output))
	copy(cp, mp.output)
	return cp
}

// ProcManager starts, stops, and streams output for managed project processes.
type ProcManager struct {
	mu    sync.Mutex
	procs map[string]*managedProc
}

func NewProcManager() *ProcManager {
	return &ProcManager{procs: make(map[string]*managedProc)}
}

// Start launches p.Command for the project. Returns an error if already running
// or no command is configured.
func (pm *ProcManager) Start(p config.Project) error {
	if p.Command == "" {
		return fmt.Errorf("no command configured for %s", p.Name)
	}
	pm.mu.Lock()
	if existing, ok := pm.procs[p.Name]; ok && !existing.done {
		pm.mu.Unlock()
		return fmt.Errorf("%s is already running", p.Name)
	}
	mp := &managedProc{}
	pm.procs[p.Name] = mp
	pm.mu.Unlock()

	cmd := exec.Command("/bin/bash", "-c", p.Command)
	if p.Dir != "" {
		cmd.Dir = p.Dir
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	mp.cmd = cmd
	mp.appendLine(fmt.Sprintf("▶ Started: %s", p.Command))

	pipe := func(r io.Reader) {
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			mp.appendLine(sc.Text())
		}
	}
	go pipe(stdout)
	go pipe(stderr)

	go func() {
		_ = cmd.Wait()
		mp.mu.Lock()
		mp.done = true
		mp.mu.Unlock()
		mp.appendLine("■ Process exited")
	}()

	return nil
}

// Stop sends SIGTERM to the named project's process.
func (pm *ProcManager) Stop(name string) error {
	pm.mu.Lock()
	mp, ok := pm.procs[name]
	pm.mu.Unlock()
	if !ok {
		return fmt.Errorf("%s is not running", name)
	}
	mp.mu.Lock()
	done, cmd := mp.done, mp.cmd
	mp.mu.Unlock()
	if done || cmd == nil || cmd.Process == nil {
		return fmt.Errorf("%s is not running", name)
	}
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		_ = cmd.Process.Kill()
	}
	return nil
}

// IsRunning returns true when the project process is active.
func (pm *ProcManager) IsRunning(name string) bool {
	pm.mu.Lock()
	mp, ok := pm.procs[name]
	pm.mu.Unlock()
	if !ok {
		return false
	}
	mp.mu.Lock()
	defer mp.mu.Unlock()
	return !mp.done
}

// GetOutput returns the buffered stdout/stderr lines for name.
func (pm *ProcManager) GetOutput(name string) []string {
	pm.mu.Lock()
	mp, ok := pm.procs[name]
	pm.mu.Unlock()
	if !ok {
		return nil
	}
	return mp.getOutput()
}

// StopAll sends SIGTERM to every managed process.
func (pm *ProcManager) StopAll() {
	pm.mu.Lock()
	names := make([]string, 0, len(pm.procs))
	for name := range pm.procs {
		names = append(names, name)
	}
	pm.mu.Unlock()
	for _, name := range names {
		_ = pm.Stop(name)
	}
}
