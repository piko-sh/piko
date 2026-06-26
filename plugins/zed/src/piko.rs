// Copyright 2026 PolitePixels Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// This project stands against fascism, authoritarianism, and all forms of
// oppression. We built this to empower people, not to enable those who would
// strip others of their rights and dignity.

use std::fs;

use zed_extension_api::{
    self as zed, settings::LspSettings, Architecture, Os, Result,
};

const LANGUAGE_SERVER_BINARY: &str = "pikopls";
const GITHUB_REPOSITORY: &str = "piko-sh/piko";

struct PikoExtension {
    cached_binary_path: Option<String>,
}

impl PikoExtension {
    fn language_server_binary_path(
        &mut self,
        language_server_id: &zed::LanguageServerId,
        worktree: &zed::Worktree,
    ) -> Result<String> {
        if let Ok(settings) = LspSettings::for_worktree(LANGUAGE_SERVER_BINARY, worktree) {
            if let Some(binary) = settings.binary {
                if let Some(path) = binary.path {
                    return Ok(path);
                }
            }
        }

        if let Some(path) = worktree.which(LANGUAGE_SERVER_BINARY) {
            return Ok(path);
        }

        if let Some(path) = &self.cached_binary_path {
            if fs::metadata(path).is_ok_and(|stat| stat.is_file()) {
                return Ok(path.clone());
            }
        }

        self.download_language_server(language_server_id)
    }

    fn download_language_server(
        &mut self,
        language_server_id: &zed::LanguageServerId,
    ) -> Result<String> {
        zed::set_language_server_installation_status(
            language_server_id,
            &zed::LanguageServerInstallationStatus::CheckingForUpdate,
        );

        let release = zed::latest_github_release(
            GITHUB_REPOSITORY,
            zed::GithubReleaseOptions {
                require_assets: true,
                pre_release: false,
            },
        )?;

        let (platform, arch) = zed::current_platform();
        let asset_name = release_asset_name(&release.version, platform, arch)?;

        let asset = release
            .assets
            .iter()
            .find(|asset| asset.name == asset_name)
            .ok_or_else(|| format!("no release asset named {asset_name}"))?;

        let version_directory = format!("{LANGUAGE_SERVER_BINARY}-{}", release.version);
        let binary_path = format!("{version_directory}/{}", binary_file_name(platform));

        if !fs::metadata(&binary_path).is_ok_and(|stat| stat.is_file()) {
            zed::set_language_server_installation_status(
                language_server_id,
                &zed::LanguageServerInstallationStatus::Downloading,
            );

            zed::download_file(
                &asset.download_url,
                &version_directory,
                downloaded_file_type(platform),
            )
            .map_err(|error| format!("failed to download {asset_name}: {error}"))?;

            zed::make_file_executable(&binary_path)?;

            remove_stale_versions(&version_directory);
        }

        if !fs::metadata(&binary_path).is_ok_and(|stat| stat.is_file()) {
            return Err(format!(
                "downloaded archive did not contain a binary at {binary_path}"
            ));
        }

        self.cached_binary_path = Some(binary_path.clone());
        Ok(binary_path)
    }
}

impl zed::Extension for PikoExtension {
    fn new() -> Self {
        Self {
            cached_binary_path: None,
        }
    }

    fn language_server_command(
        &mut self,
        language_server_id: &zed::LanguageServerId,
        worktree: &zed::Worktree,
    ) -> Result<zed::Command> {
        let binary_path = self.language_server_binary_path(language_server_id, worktree)?;

        let mut arguments = LspSettings::for_worktree(LANGUAGE_SERVER_BINARY, worktree)
            .ok()
            .and_then(|settings| settings.binary)
            .and_then(|binary| binary.arguments)
            .unwrap_or_default();

        if !arguments.iter().any(|argument| argument.starts_with("--gopls-bridge")) {
            arguments.push("--gopls-bridge=true".to_string());
        }
        if !arguments.iter().any(|argument| argument.starts_with("--gopls-path")) {
            if let Some(gopls_path) = worktree.which("gopls") {
                arguments.push(format!("--gopls-path={gopls_path}"));
            }
        }

        Ok(zed::Command {
            command: binary_path,
            args: arguments,
            env: worktree.shell_env(),
        })
    }
}

fn release_asset_name(version: &str, platform: Os, arch: Architecture) -> Result<String> {
    let version = version.strip_prefix('v').unwrap_or(version);
    let os = os_token(platform);
    let architecture = arch_token(arch)?;
    let extension = archive_extension(platform);
    Ok(format!(
        "{LANGUAGE_SERVER_BINARY}-{version}-{os}-{architecture}.{extension}"
    ))
}

fn os_token(platform: Os) -> &'static str {
    match platform {
        Os::Mac => "darwin",
        Os::Linux => "linux",
        Os::Windows => "windows",
    }
}

fn arch_token(arch: Architecture) -> Result<&'static str> {
    match arch {
        Architecture::Aarch64 => Ok("arm64"),
        Architecture::X8664 => Ok("amd64"),
        Architecture::X86 => Err("pikopls is not built for 32-bit x86".into()),
    }
}

fn archive_extension(platform: Os) -> &'static str {
    match platform {
        Os::Windows => "zip",
        Os::Mac | Os::Linux => "tar.gz",
    }
}

fn downloaded_file_type(platform: Os) -> zed::DownloadedFileType {
    match platform {
        Os::Windows => zed::DownloadedFileType::Zip,
        Os::Mac | Os::Linux => zed::DownloadedFileType::GzipTar,
    }
}

fn binary_file_name(platform: Os) -> String {
    match platform {
        Os::Windows => format!("{LANGUAGE_SERVER_BINARY}.exe"),
        Os::Mac | Os::Linux => LANGUAGE_SERVER_BINARY.to_string(),
    }
}

fn remove_stale_versions(current_directory: &str) {
    let Ok(entries) = fs::read_dir(".") else {
        return;
    };
    for entry in entries.flatten() {
        let name = entry.file_name();
        let name = name.to_string_lossy();
        if name.starts_with(LANGUAGE_SERVER_BINARY) && name != current_directory {
            let _ = fs::remove_dir_all(entry.path());
        }
    }
}

zed::register_extension!(PikoExtension);
