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
    self as zed,
    settings::{CommandSettings, LspSettings},
    Architecture, LanguageServerInstallationStatus, Os, Result,
};

const LANGUAGE_SERVER_BINARY: &str = "pikopls";
const GITHUB_REPOSITORY: &str = "piko-sh/piko";
const GOPLS_BRIDGE_FLAG: &str = "gopls-bridge";
const GOPLS_DISABLE_FLAG: &str = "gopls-disable";
const GOPLS_PATH_FLAG: &str = "gopls-path";
const GOPLS_PATH_ENV: &str = "PIKO_GOPLS_PATH";
const VERSION_DIRECTORY_PREFIX: &str = "pikopls-";

struct PikoExtension {
    cached_binary_path: Option<String>,
}

impl PikoExtension {
    fn language_server_binary_path(
        &mut self,
        language_server_id: &zed::LanguageServerId,
        worktree: &zed::Worktree,
        binary_settings: Option<&CommandSettings>,
    ) -> Result<String> {
        if let Some(path) = binary_settings.and_then(|binary| binary.path.clone()) {
            return Ok(path);
        }

        if let Some(path) = worktree.which(LANGUAGE_SERVER_BINARY) {
            return Ok(path);
        }

        if let Some(path) = self.cached_binary_path.clone() {
            if is_file(&path) {
                zed::make_file_executable(&path)
                    .map_err(|error| format!("failed to make {path} executable: {error}"))?;
                return Ok(path);
            }
        }

        match self.download_language_server(language_server_id) {
            Ok(path) => {
                zed::set_language_server_installation_status(
                    language_server_id,
                    &LanguageServerInstallationStatus::None,
                );
                Ok(path)
            }
            Err(error) => {
                let message =
                    format!("{LANGUAGE_SERVER_BINARY} not on PATH and download failed: {error}");
                zed::set_language_server_installation_status(
                    language_server_id,
                    &LanguageServerInstallationStatus::Failed(message.clone()),
                );
                Err(message)
            }
        }
    }

    fn download_language_server(
        &mut self,
        language_server_id: &zed::LanguageServerId,
    ) -> Result<String> {
        zed::set_language_server_installation_status(
            language_server_id,
            &LanguageServerInstallationStatus::CheckingForUpdate,
        );

        let (platform, arch) = zed::current_platform();

        let release = match latest_release() {
            Ok(release) => release,
            Err(error) => {
                return self
                    .reuse_installed_binary(platform)
                    .ok_or_else(|| format!("release lookup failed: {error}"));
            }
        };

        let asset_name = release_asset_name(&release.version, platform, arch)?;

        let asset = release
            .assets
            .iter()
            .find(|asset| asset.name == asset_name)
            .ok_or_else(|| format!("no release asset named {asset_name}"))?;

        let version_directory = format!("{VERSION_DIRECTORY_PREFIX}{}", release.version);
        let binary_path = format!("{version_directory}/{}", binary_file_name(platform));

        if !is_file(&binary_path) {
            zed::set_language_server_installation_status(
                language_server_id,
                &LanguageServerInstallationStatus::Downloading,
            );

            zed::download_file(
                &asset.download_url,
                &version_directory,
                downloaded_file_type(platform),
            )
            .map_err(|error| format!("failed to download {asset_name}: {error}"))?;

            remove_stale_versions(&version_directory);
        }

        if !is_file(&binary_path) {
            return Err(format!(
                "downloaded archive did not contain a binary at {binary_path}"
            ));
        }

        zed::make_file_executable(&binary_path)
            .map_err(|error| format!("failed to make {binary_path} executable: {error}"))?;
        self.cached_binary_path = Some(binary_path.clone());
        Ok(binary_path)
    }

    fn reuse_installed_binary(&mut self, platform: Os) -> Option<String> {
        let binary_name = binary_file_name(platform);
        let entries = match fs::read_dir(".") {
            Ok(entries) => entries,
            Err(error) => {
                eprintln!("{LANGUAGE_SERVER_BINARY}: cannot list installed versions: {error}");
                return None;
            }
        };
        let newest = entries
            .flatten()
            .filter_map(|entry| {
                let name = entry.file_name().to_string_lossy().into_owned();
                let version = name.strip_prefix(VERSION_DIRECTORY_PREFIX)?;
                let path = format!("{name}/{binary_name}");
                is_file(&path).then(|| (version_key(version), path))
            })
            .max_by(|(left, _), (right, _)| left.cmp(right));

        let (_, path) = newest?;
        if let Err(error) = zed::make_file_executable(&path) {
            eprintln!("{LANGUAGE_SERVER_BINARY}: cannot make {path} executable: {error}");
            return None;
        }
        self.cached_binary_path = Some(path.clone());
        Some(path)
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
        let binary_settings = match LspSettings::for_worktree(LANGUAGE_SERVER_BINARY, worktree) {
            Ok(settings) => settings.binary,
            Err(error) => {
                eprintln!(
                    "{LANGUAGE_SERVER_BINARY}: cannot read LSP settings, using defaults: {error}"
                );
                None
            }
        };

        let binary_path = self.language_server_binary_path(
            language_server_id,
            worktree,
            binary_settings.as_ref(),
        )?;

        let (user_arguments, user_env) = match binary_settings {
            Some(binary) => (
                binary.arguments.unwrap_or_default(),
                binary.env.unwrap_or_default(),
            ),
            None => Default::default(),
        };

        let mut env = worktree.shell_env();
        for (key, value) in user_env {
            match env.iter_mut().find(|(existing, _)| *existing == key) {
                Some(entry) => entry.1 = value,
                None => env.push((key, value)),
            }
        }

        let env_names_gopls = environment_names_gopls(&env);
        let arguments = gopls_arguments(user_arguments, || {
            if env_names_gopls {
                None
            } else {
                worktree.which("gopls")
            }
        });

        Ok(zed::Command {
            command: binary_path,
            args: arguments,
            env,
        })
    }
}

fn latest_release() -> Result<zed::GithubRelease> {
    zed::latest_github_release(
        GITHUB_REPOSITORY,
        zed::GithubReleaseOptions {
            require_assets: true,
            pre_release: false,
        },
    )
    .or_else(|stable_error| {
        zed::latest_github_release(
            GITHUB_REPOSITORY,
            zed::GithubReleaseOptions {
                require_assets: true,
                pre_release: true,
            },
        )
        .map_err(|pre_release_error| {
            format!("stable release: {stable_error}; pre-release: {pre_release_error}")
        })
    })
}

fn environment_names_gopls(env: &[(String, String)]) -> bool {
    env.iter()
        .any(|(key, value)| key == GOPLS_PATH_ENV && !value.is_empty())
}

fn gopls_arguments(
    mut arguments: Vec<String>,
    find_gopls: impl FnOnce() -> Option<String>,
) -> Vec<String> {
    let has_flag = |name: &str| {
        arguments
            .iter()
            .any(|argument| flag_name(argument) == Some(name))
    };
    let bridge_disabled = arguments.iter().any(|argument| {
        flag_name(argument) == Some(GOPLS_DISABLE_FLAG)
            && !matches!(argument.split_once('='), Some((_, "false" | "0")))
    });

    let bridge_configured = bridge_disabled || has_flag(GOPLS_BRIDGE_FLAG);
    let path_configured = has_flag(GOPLS_PATH_FLAG);

    if !bridge_configured {
        arguments.push(format!("--{GOPLS_BRIDGE_FLAG}=true"));
    }
    if !path_configured && !bridge_disabled {
        if let Some(gopls_path) = find_gopls() {
            arguments.push(format!("--{GOPLS_PATH_FLAG}={gopls_path}"));
        }
    }
    arguments
}

fn flag_name(argument: &str) -> Option<&str> {
    let trimmed = argument.trim_start_matches('-');
    if trimmed.len() == argument.len() {
        return None;
    }
    Some(trimmed.split_once('=').map_or(trimmed, |(name, _)| name))
}

fn is_file(path: &str) -> bool {
    fs::metadata(path).is_ok_and(|stat| stat.is_file())
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

fn version_key(version: &str) -> Vec<u64> {
    version
        .trim_start_matches('v')
        .split(['.', '-', '+'])
        .map(|part| part.parse().unwrap_or(0))
        .collect()
}

fn remove_stale_versions(current_directory: &str) {
    let entries = match fs::read_dir(".") {
        Ok(entries) => entries,
        Err(error) => {
            eprintln!("{LANGUAGE_SERVER_BINARY}: cannot list old versions to remove: {error}");
            return;
        }
    };
    for entry in entries.flatten() {
        let name = entry.file_name();
        let name = name.to_string_lossy();
        if name.starts_with(VERSION_DIRECTORY_PREFIX) && name != current_directory {
            if let Err(error) = fs::remove_dir_all(entry.path()) {
                eprintln!("{LANGUAGE_SERVER_BINARY}: cannot remove old version {name}: {error}");
            }
        }
    }
}

zed::register_extension!(PikoExtension);

#[cfg(test)]
mod tests {
    use super::{flag_name, gopls_arguments, version_key};

    fn strings(values: &[&str]) -> Vec<String> {
        values.iter().map(ToString::to_string).collect()
    }

    #[test]
    fn flag_name_strips_any_number_of_dashes_and_values() {
        assert_eq!(flag_name("-gopls-disable"), Some("gopls-disable"));
        assert_eq!(flag_name("--gopls-disable"), Some("gopls-disable"));
        assert_eq!(flag_name("---gopls-bridge=false"), Some("gopls-bridge"));
        assert_eq!(flag_name("gopls-bridge"), None);
    }

    #[test]
    fn bridge_and_path_are_added_by_default() {
        let arguments = gopls_arguments(Vec::new(), || Some("/bin/gopls".to_string()));
        assert_eq!(
            arguments,
            strings(&["--gopls-bridge=true", "--gopls-path=/bin/gopls"])
        );
    }

    #[test]
    fn single_dash_bridge_opt_out_is_honoured() {
        let arguments = gopls_arguments(strings(&["-gopls-bridge=false"]), || {
            Some("/bin/gopls".to_string())
        });
        assert_eq!(
            arguments,
            strings(&["-gopls-bridge=false", "--gopls-path=/bin/gopls"])
        );
    }

    #[test]
    fn gopls_disable_suppresses_bridge_and_path() {
        let arguments = gopls_arguments(strings(&["--gopls-disable"]), || {
            Some("/bin/gopls".to_string())
        });
        assert_eq!(arguments, strings(&["--gopls-disable"]));
    }

    #[test]
    fn gopls_disable_false_keeps_the_bridge() {
        let arguments = gopls_arguments(strings(&["--gopls-disable=false"]), || None);
        assert_eq!(
            arguments,
            strings(&["--gopls-disable=false", "--gopls-bridge=true"])
        );
    }

    #[test]
    fn version_key_orders_versions_numerically() {
        assert!(version_key("0.10.0") > version_key("0.9.0"));
        assert!(version_key("v1.2.3") == version_key("1.2.3"));
        assert!(version_key("1.2.0") > version_key("1.1.99"));
        assert_eq!(version_key("1.2.x"), vec![1, 2, 0]);
    }

    #[test]
    fn user_gopls_path_is_kept() {
        let arguments = gopls_arguments(strings(&["-gopls-path=/opt/gopls"]), || {
            Some("/bin/gopls".to_string())
        });
        assert_eq!(
            arguments,
            strings(&["-gopls-path=/opt/gopls", "--gopls-bridge=true"])
        );
    }
}
