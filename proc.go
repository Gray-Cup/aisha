package main

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"syscall"
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

type procManager struct {
	mu    sync.Mutex
	procs map[string]*managedProc
}

func newProcManager() *procManager {
	return &procManager{procs: make(map[string]*managedProc)}
}

func (pm *procManager) start(p Project) error {
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

	pipeLines := func(r io.Reader) {
		sc := bufio.NewScanner(r)
		for sc.Scan() {
			mp.appendLine(sc.Text())
		}
	}
	go pipeLines(stdout)
	go pipeLines(stderr)

	go func() {
		_ = cmd.Wait()
		mp.mu.Lock()
		mp.done = true
		mp.mu.Unlock()
		mp.appendLine("■ Process exited")
	}()

	return nil
}

func (pm *procManager) stop(name string) error {
	pm.mu.Lock()
	mp, ok := pm.procs[name]
	pm.mu.Unlock()
	if !ok {
		return fmt.Errorf("%s is not running", name)
	}
	mp.mu.Lock()
	done := mp.done
	cmd := mp.cmd
	mp.mu.Unlock()
	if done || cmd == nil || cmd.Process == nil {
		return fmt.Errorf("%s is not running", name)
	}
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		_ = cmd.Process.Kill()
	}
	return nil
}

func (pm *procManager) isRunning(name string) bool {
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

func (pm *procManager) getOutput(name string) []string {
	pm.mu.Lock()
	mp, ok := pm.procs[name]
	pm.mu.Unlock()
	if !ok {
		return nil
	}
	return mp.getOutput()
}

func (pm *procManager) stopAll() {
	pm.mu.Lock()
	names := make([]string, 0, len(pm.procs))
	for name := range pm.procs {
		names = append(names, name)
	}
	pm.mu.Unlock()
	for _, name := range names {
		_ = pm.stop(name)
	}
}
