import type { AppInfo, Project } from "./types";

const BASE = "";  // proxied via Vite in dev; same origin in prod

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(BASE + path, init);
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error((body as { error?: string }).error ?? `HTTP ${res.status}`);
  }
  if (res.status === 204) return undefined as T;
  return res.json() as Promise<T>;
}

export const api = {
  info: () => req<AppInfo>("/api/info"),

  listProjects: () => req<Project[]>("/api/projects"),

  createProject: (name: string, command: string, cwd: string, port = 0) =>
    req<Project>("/api/projects", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name, command, cwd, port }),
    }),

  startProject: (id: string) =>
    req<Project>(`/api/projects/${id}/start`, { method: "POST" }),

  stopProject: (id: string) =>
    req<Project>(`/api/projects/${id}/stop`, { method: "POST" }),

  deleteProject: (id: string) =>
    req<void>(`/api/projects/${id}`, { method: "DELETE" }),

  getLogs: (id: string) =>
    fetch(`/api/projects/${id}/logs`).then((r) => r.text()),
};
