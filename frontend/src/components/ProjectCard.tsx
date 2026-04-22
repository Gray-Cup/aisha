import type { Project } from "../types";

interface Props {
  project: Project;
  lanIp: string;
  gatewayPort: string;
  onStart: () => void;
  onStop: () => void;
  onDelete: () => void;
  onLogs: () => void;
  loading: boolean;
}

export function ProjectCard({ project, lanIp, gatewayPort, onStart, onStop, onDelete, onLogs, loading }: Props) {
  const running = project.Status === "running";
  const wildcardUrl = `http://${project.ID}.localhost:${gatewayPort}`;
  const pathUrl = `http://localhost:${gatewayPort}/${project.ID}`;
  const lanUrl = `http://${lanIp}:${gatewayPort}/${project.ID}`;

  return (
    <div className={`card ${running ? "card-running" : ""}`}>
      <div className="card-header">
        <div>
          <span className={`status-dot ${running ? "dot-green" : "dot-gray"}`} />
          <strong>{project.Name}</strong>
          <span className="badge">:{project.Port}</span>
        </div>
        <span className={`tag ${running ? "tag-green" : "tag-gray"}`}>
          {project.Status}
        </span>
      </div>

      <p className="card-cmd">{project.Command}</p>
      <p className="card-cwd muted">{project.CWD}</p>

      {running && (
        <div className="card-urls">
          <div className="url-row">
            <span className="url-label">wildcard</span>
            <a href={wildcardUrl} target="_blank" rel="noreferrer">{wildcardUrl}</a>
          </div>
          <div className="url-row">
            <span className="url-label">local</span>
            <a href={pathUrl} target="_blank" rel="noreferrer">{pathUrl}</a>
          </div>
          <div className="url-row">
            <span className="url-label">lan</span>
            <a href={lanUrl} target="_blank" rel="noreferrer">{lanUrl}</a>
          </div>
        </div>
      )}

      <div className="card-actions">
        {running ? (
          <button className="btn-danger" onClick={onStop} disabled={loading}>Stop</button>
        ) : (
          <button className="btn-primary" onClick={onStart} disabled={loading}>Start</button>
        )}
        <button className="btn-secondary" onClick={onLogs}>Logs</button>
        <button className="btn-ghost" onClick={onDelete} disabled={loading}>Delete</button>
      </div>
    </div>
  );
}
