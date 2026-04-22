import { useState } from "react";

interface Props {
  onClose: () => void;
  onCreate: (name: string, command: string, cwd: string, port: number) => Promise<void>;
}

export function CreateModal({ onClose, onCreate }: Props) {
  const [name, setName] = useState("");
  const [command, setCommand] = useState("npm run dev");
  const [cwd, setCwd] = useState("");
  const [port, setPort] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    const portNum = port.trim() ? parseInt(port.trim(), 10) : 0;
    if (port.trim() && (isNaN(portNum) || portNum < 1 || portNum > 65535)) {
      setError("Port must be a number between 1 and 65535");
      return;
    }
    setLoading(true);
    try {
      await onCreate(name.trim(), command.trim(), cwd.trim(), portNum);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <h2>New Project</h2>
        <form onSubmit={submit}>
          <label>
            Name
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="my-app"
              required
              autoFocus
            />
          </label>
          <label>
            Start command
            <input
              value={command}
              onChange={(e) => setCommand(e.target.value)}
              placeholder="npm run dev"
              required
            />
          </label>
          <label>
            Working directory
            <input
              value={cwd}
              onChange={(e) => setCwd(e.target.value)}
              placeholder="/Users/you/projects/my-app"
              required
            />
          </label>
          <label>
            App port
            <div className="input-with-hint">
              <input
                value={port}
                onChange={(e) => setPort(e.target.value)}
                placeholder="auto-assign"
                inputMode="numeric"
                pattern="[0-9]*"
              />
              <span className="input-hint">leave blank to auto-assign (4000–4999)</span>
            </div>
          </label>
          {error && <p className="error">{error}</p>}
          <div className="modal-actions">
            <button type="button" className="btn-secondary" onClick={onClose}>
              Cancel
            </button>
            <button type="submit" className="btn-primary" disabled={loading}>
              {loading ? "Creating…" : "Create"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
