package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"aisha/backend/db"
)

type Manager struct {
	db        *db.DB
	dataDir   string
	ports     *portAllocator
	processes map[string]*process
	mu        sync.RWMutex
}

func NewManager(database *db.DB, dataDir string) *Manager {
	return &Manager{
		db:        database,
		dataDir:   dataDir,
		processes: make(map[string]*process),
	}
}

// RestoreState seeds the port allocator from DB and marks running projects as stopped.
func (m *Manager) RestoreState() error {
	used, err := m.db.GetUsedPorts()
	if err != nil {
		return err
	}
	m.ports = newPortAllocator(used)

	projects, err := m.db.ListProjects()
	if err != nil {
		return err
	}
	for _, p := range projects {
		if p.Status == "running" {
			m.db.UpdateProjectStatus(p.ID, "stopped")
		}
	}
	return nil
}

// CreateProject creates a new project. If fixedPort > 0 it is used as-is;
// otherwise a free port is auto-assigned from the 4000–4999 range.
func (m *Manager) CreateProject(name, command, cwd string, fixedPort int) (*db.Project, error) {
	if _, err := os.Stat(cwd); err != nil {
		return nil, fmt.Errorf("directory %q not found", cwd)
	}

	id := toID(name)
	if id == "" {
		return nil, fmt.Errorf("invalid project name")
	}

	if _, err := m.db.GetProject(id); err == nil {
		return nil, fmt.Errorf("project %q already exists", id)
	}

	var port int
	if fixedPort > 0 {
		port = fixedPort
		m.ports.Reserve(port)
	} else {
		var err error
		port, err = m.ports.Allocate()
		if err != nil {
			return nil, err
		}
	}

	proj := db.Project{
		ID:        id,
		Name:      name,
		Port:      port,
		Status:    "stopped",
		Command:   command,
		CWD:       cwd,
		CreatedAt: time.Now(),
	}

	if err := m.db.InsertProject(proj); err != nil {
		m.ports.Free(port)
		return nil, err
	}

	return &proj, nil
}

func (m *Manager) StartProject(id string) error {
	proj, err := m.db.GetProject(id)
	if err != nil {
		return fmt.Errorf("project %q not found", id)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if proc, ok := m.processes[id]; ok && proc.running() {
		return fmt.Errorf("project %q is already running", id)
	}

	logFile := filepath.Join(m.dataDir, "logs", id+".log")
	proc, err := startProcess(proj.Command, proj.CWD, logFile, proj.Port)
	if err != nil {
		return err
	}

	m.processes[id] = proc
	m.db.UpdateProjectStatus(id, "running")

	go func() {
		<-proc.wait()
		m.mu.Lock()
		delete(m.processes, id)
		m.mu.Unlock()
		m.db.UpdateProjectStatus(id, "stopped")
	}()

	return nil
}

func (m *Manager) StopProject(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	proc, ok := m.processes[id]
	if !ok {
		return fmt.Errorf("project %q is not running", id)
	}

	if err := proc.stop(); err != nil {
		return err
	}
	delete(m.processes, id)
	m.db.UpdateProjectStatus(id, "stopped")
	return nil
}

func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, proc := range m.processes {
		proc.stop()
		delete(m.processes, id)
		m.db.UpdateProjectStatus(id, "stopped")
	}
}

func (m *Manager) DeleteProject(id string) error {
	m.mu.Lock()
	if proc, ok := m.processes[id]; ok {
		proc.stop()
		delete(m.processes, id)
	}
	m.mu.Unlock()

	proj, err := m.db.GetProject(id)
	if err != nil {
		return fmt.Errorf("project %q not found", id)
	}
	m.ports.Free(proj.Port)
	return m.db.DeleteProject(id)
}

func (m *Manager) GetProject(id string) (*db.Project, error) {
	return m.db.GetProject(id)
}

func (m *Manager) ListProjects() ([]db.Project, error) {
	return m.db.ListProjects()
}

func (m *Manager) GetPort(id string) (int, error) {
	proj, err := m.db.GetProject(id)
	if err != nil {
		return 0, err
	}
	return proj.Port, nil
}

func toID(name string) string {
	var b strings.Builder
	for _, c := range strings.ToLower(name) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			b.WriteRune(c)
		} else if b.Len() > 0 {
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}
