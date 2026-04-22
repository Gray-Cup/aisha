import { useEffect, useRef, useState } from "react";
import { api } from "../api";
import type { Project } from "../types";

interface Props {
  project: Project;
  onClose: () => void;
}

export function LogsModal({ project, onClose }: Props) {
  const [logs, setLogs] = useState<string>("");
  const [loading, setLoading] = useState(true);
  const bottomRef = useRef<HTMLDivElement>(null);

  async function fetchLogs() {
    try {
      const text = await api.getLogs(project.ID);
      setLogs(text);
    } catch {
      setLogs("(failed to load logs)");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    fetchLogs();
    const interval = setInterval(fetchLogs, 2000);
    return () => clearInterval(interval);
  }, [project.ID]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [logs]);

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal logs-modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>Logs — {project.Name}</h2>
          <button className="btn-secondary" onClick={onClose}>Close</button>
        </div>
        <div className="log-body">
          {loading ? (
            <span className="muted">Loading…</span>
          ) : logs ? (
            <pre>{logs}</pre>
          ) : (
            <span className="muted">No logs yet.</span>
          )}
          <div ref={bottomRef} />
        </div>
      </div>
    </div>
  );
}
