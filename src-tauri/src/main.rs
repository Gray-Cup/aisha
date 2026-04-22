#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use std::sync::Mutex;
use tauri::api::process::{Command, CommandChild};
use tauri::{Manager, RunEvent};

struct BackendProcess(Mutex<Option<CommandChild>>);

fn main() {
    tauri::Builder::default()
        .setup(|app| {
            // Resolve the data directory for the Go backend.
            let data_dir = app
                .path_resolver()
                .app_data_dir()
                .expect("failed to resolve app data dir");

            std::fs::create_dir_all(&data_dir).ok();

            let (mut _rx, child) = Command::new_sidecar("aisha-backend")
                .expect("aisha-backend sidecar not found — run `make build-backend` first")
                .envs([("TWISHA_DATA_DIR", data_dir.to_str().unwrap_or(""))])
                .spawn()
                .expect("failed to spawn Go backend");

            app.manage(BackendProcess(Mutex::new(Some(child))));
            Ok(())
        })
        .build(tauri::generate_context!())
        .expect("error building Tauri application")
        .run(|app_handle, event| {
            if let RunEvent::Exit = event {
                if let Some(child) = app_handle
                    .state::<BackendProcess>()
                    .0
                    .lock()
                    .unwrap()
                    .take()
                {
                    child.kill().ok();
                }
            }
        });
}
