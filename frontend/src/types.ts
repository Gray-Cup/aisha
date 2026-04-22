export interface Project {
  ID: string;
  Name: string;
  Port: number;
  Status: "running" | "stopped";
  Command: string;
  CWD: string;
  CreatedAt: string;
}

export interface AppInfo {
  ip: string;
  port: string;
  base_url: string;
  lan_url: string;
}
