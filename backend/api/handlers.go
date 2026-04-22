package api

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"strings"

	"aisha/backend/db"
	"aisha/backend/logs"
)

type Manager interface {
	CreateProject(name, command, cwd string, fixedPort int) (*db.Project, error)
	StartProject(id string) error
	StopProject(id string) error
	GetProject(id string) (*db.Project, error)
	ListProjects() ([]db.Project, error)
	DeleteProject(id string) error
}

func RegisterHandlers(mux *http.ServeMux, mgr Manager, dataDir string) {
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listProjects(w, mgr)
		case http.MethodPost:
			createProject(w, r, mgr)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/projects/", func(w http.ResponseWriter, r *http.Request) {
		trimmed := strings.TrimPrefix(r.URL.Path, "/api/projects/")
		parts := strings.SplitN(trimmed, "/", 2)
		id := parts[0]
		action := ""
		if len(parts) == 2 {
			action = parts[1]
		}

		switch action {
		case "":
			switch r.Method {
			case http.MethodGet:
				getProject(w, r, mgr, id)
			case http.MethodDelete:
				deleteProject(w, r, mgr, id)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
		case "start":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			startProject(w, mgr, id)
		case "stop":
			if r.Method != http.MethodPost {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			stopProject(w, mgr, id)
		case "logs":
			getProjectLogs(w, dataDir, id)
		default:
			http.NotFound(w, r)
		}
	})

	mux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		ip := localIP()
		port := os.Getenv("AISHA_PORT")
		if port == "" {
			port = "3000"
		}
		respond(w, http.StatusOK, map[string]string{
			"ip":       ip,
			"port":     port,
			"base_url": "http://localhost:" + port,
			"lan_url":  "http://" + ip + ":" + port,
		})
	})
}

func listProjects(w http.ResponseWriter, mgr Manager) {
	projects, err := mgr.ListProjects()
	if err != nil {
		respond(w, http.StatusInternalServerError, errBody(err))
		return
	}
	if projects == nil {
		projects = []db.Project{}
	}
	respond(w, http.StatusOK, projects)
}

func createProject(w http.ResponseWriter, r *http.Request, mgr Manager) {
	var req struct {
		Name    string `json:"name"`
		Command string `json:"command"`
		CWD     string `json:"cwd"`
		Port    int    `json:"port"` // 0 = auto-assign
	}
	body, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(body, &req); err != nil {
		respond(w, http.StatusBadRequest, errBody(err))
		return
	}
	if req.Name == "" || req.Command == "" || req.CWD == "" {
		respond(w, http.StatusBadRequest, map[string]string{"error": "name, command, and cwd are required"})
		return
	}
	proj, err := mgr.CreateProject(req.Name, req.Command, req.CWD, req.Port)
	if err != nil {
		respond(w, http.StatusBadRequest, errBody(err))
		return
	}
	respond(w, http.StatusCreated, proj)
}

func getProject(w http.ResponseWriter, r *http.Request, mgr Manager, id string) {
	proj, err := mgr.GetProject(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	respond(w, http.StatusOK, proj)
}

func deleteProject(w http.ResponseWriter, r *http.Request, mgr Manager, id string) {
	if err := mgr.DeleteProject(id); err != nil {
		respond(w, http.StatusBadRequest, errBody(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func startProject(w http.ResponseWriter, mgr Manager, id string) {
	if err := mgr.StartProject(id); err != nil {
		respond(w, http.StatusBadRequest, errBody(err))
		return
	}
	proj, _ := mgr.GetProject(id)
	respond(w, http.StatusOK, proj)
}

func stopProject(w http.ResponseWriter, mgr Manager, id string) {
	if err := mgr.StopProject(id); err != nil {
		respond(w, http.StatusBadRequest, errBody(err))
		return
	}
	proj, _ := mgr.GetProject(id)
	respond(w, http.StatusOK, proj)
}

func getProjectLogs(w http.ResponseWriter, dataDir, id string) {
	content, err := logs.ReadAll(dataDir, id)
	if err != nil {
		respond(w, http.StatusInternalServerError, errBody(err))
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(content))
}

func respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func errBody(err error) map[string]string {
	return map[string]string{"error": err.Error()}
}

func localIP() string {
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}
	return "127.0.0.1"
}
