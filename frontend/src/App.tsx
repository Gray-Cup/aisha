import { useCallback, useEffect, useState } from "react";
import { api } from "./api";
import { CreateModal } from "./components/CreateModal";
import { LogsModal } from "./components/LogsModal";
import { ProjectCard } from "./components/ProjectCard";
import type { AppInfo, Project } from "./types";
import "./index.css";

export default function App() {
  const [projects, setProjects] = useState<Project[]>([]);
  const [info, setInfo] = useState<AppInfo | null>(null);
  const [showCreate, setShowCreate] = useState(false);
  const [logsProject, setLogsProject] = useState<Project | null>(null);
  const [loadingId, setLoadingId] = useState<string | null>(null);
  const [error, setError] = useState("");

  const refresh = useCallback(async () => {
    try {
      const [ps, inf] = await Promise.all([api.listProjects(), api.info()]);
      setProjects(ps);
      setInfo(inf);
    } catch {
      /* backend not ready yet */
    }
  }, []);

  useEffect(() => {
    refresh();
    const t = setInterval(refresh, 3000);
    return () => clearInterval(t);
  }, [refresh]);

  async function withLoading(id: string, fn: () => Promise<void>) {
    setLoadingId(id);
    setError("");
    try {
      await fn();
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Unknown error");
    } finally {
      setLoadingId(null);
    }
  }

  async function handleCreate(name: string, command: string, cwd: string, port: number) {
    await api.createProject(name, command, cwd, port);
    await refresh();
  }

  return (
    <div className="app">
      <header className="header">
        <div className="header-left">
          <span className="logo">⬡ Aisha</span>
          <span className="muted subtitle">Local project orchestrator</span>
        </div>
        <div className="header-right">
          {info && (
            <div className="info-pills">
              <span className="pill">localhost:3000</span>
              <span className="pill">{info.ip}:3000</span>
            </div>
          )}
          <button className="btn-primary" onClick={() => setShowCreate(true)}>
            + New Project
          </button>
        </div>
      </header>

      {error && (
        <div className="alert-error" onClick={() => setError("")}>
          {error}
        </div>
      )}

      <div className="routing-banner">
        <span className="routing-title">Routing</span>
        <span className="routing-item">
          <span className="url-label">wildcard</span>
          <code>project.localhost:3000</code>
          — Chrome/Firefox only, works out of the box
        </span>
        <span className="routing-item">
          <span className="url-label">path</span>
          <code>localhost:3000/project</code>
          — any browser
        </span>
        <span className="routing-item">
          <span className="url-label">lan</span>
          <code>{info?.ip ?? "…"}:3000/project</code>
          — other devices on your network
        </span>
      </div>

      <main className="main">
        {projects.length === 0 ? (
          <div className="empty">
            <p>No projects yet.</p>
            <button className="btn-primary" onClick={() => setShowCreate(true)}>
              Create your first project
            </button>
          </div>
        ) : (
          <div className="grid">
            {projects.map((p) => (
              <ProjectCard
                key={p.ID}
                project={p}
                lanIp={info?.ip ?? "localhost"}
                gatewayPort={info?.port ?? "3000"}
                loading={loadingId === p.ID}
                onStart={() => withLoading(p.ID, async () => { await api.startProject(p.ID); })}
                onStop={() => withLoading(p.ID, async () => { await api.stopProject(p.ID); })}
                onDelete={() =>
                  withLoading(p.ID, async () => {
                    if (confirm(`Delete project "${p.Name}"?`)) {
                      await api.deleteProject(p.ID);
                    }
                  })
                }
                onLogs={() => setLogsProject(p)}
              />
            ))}
          </div>
        )}
      </main>

      {showCreate && (
        <CreateModal onClose={() => setShowCreate(false)} onCreate={handleCreate} />
      )}
      {logsProject && (
        <LogsModal project={logsProject} onClose={() => setLogsProject(null)} />
      )}
    </div>
  );
}
