package db

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

type Project struct {
	ID        string    `json:"ID"`
	Name      string    `json:"Name"`
	Port      int       `json:"Port"`
	Status    string    `json:"Status"`
	Command   string    `json:"Command"`
	CWD       string    `json:"CWD"`
	CreatedAt time.Time `json:"CreatedAt"`
}

func Init(path string) (*DB, error) {
	sqldb, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	sqldb.SetMaxOpenConns(1)
	d := &DB{sqldb}
	return d, d.migrate()
}

func (d *DB) migrate() error {
	_, err := d.Exec(`CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		port INTEGER NOT NULL,
		status TEXT NOT NULL DEFAULT 'stopped',
		command TEXT NOT NULL,
		cwd TEXT NOT NULL,
		created_at DATETIME NOT NULL
	)`)
	return err
}

func (d *DB) InsertProject(p Project) error {
	_, err := d.Exec(
		`INSERT INTO projects (id, name, port, status, command, cwd, created_at) VALUES (?,?,?,?,?,?,?)`,
		p.ID, p.Name, p.Port, p.Status, p.Command, p.CWD, p.CreatedAt,
	)
	return err
}

func (d *DB) GetProject(id string) (*Project, error) {
	row := d.QueryRow(
		`SELECT id, name, port, status, command, cwd, created_at FROM projects WHERE id = ?`, id,
	)
	p := &Project{}
	return p, row.Scan(&p.ID, &p.Name, &p.Port, &p.Status, &p.Command, &p.CWD, &p.CreatedAt)
}

func (d *DB) ListProjects() ([]Project, error) {
	rows, err := d.Query(
		`SELECT id, name, port, status, command, cwd, created_at FROM projects ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Port, &p.Status, &p.Command, &p.CWD, &p.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (d *DB) UpdateProjectStatus(id, status string) error {
	_, err := d.Exec(`UPDATE projects SET status = ? WHERE id = ?`, status, id)
	return err
}

func (d *DB) GetUsedPorts() ([]int, error) {
	rows, err := d.Query(`SELECT port FROM projects`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ports []int
	for rows.Next() {
		var p int
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		ports = append(ports, p)
	}
	return ports, rows.Err()
}

func (d *DB) DeleteProject(id string) error {
	_, err := d.Exec(`DELETE FROM projects WHERE id = ?`, id)
	return err
}
