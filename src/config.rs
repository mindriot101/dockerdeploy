use anyhow::Result;
use serde::Deserialize;

#[derive(Deserialize, Debug, Default)]
pub(crate) struct DockerDeployConfig {
    pub(crate) api_version: String,
    pub(crate) validation_key: Option<String>,
    pub(crate) server: Option<ServerConfig>,
    pub(crate) image: ImageConfig,
    pub(crate) container: ContainerConfig,
    pub(crate) branch: BranchConfig,
    pub(crate) heartbeat: HeartbeatConfig,
}

impl DockerDeployConfig {
    pub(crate) fn from_file<P: AsRef<std::path::Path>>(path: P) -> Result<Self> {
        let text = std::fs::read_to_string(path)?;
        let config = toml::from_str(&text)?;
        Ok(config)
    }
}

#[derive(Deserialize, Debug, Default)]
pub(crate) struct ServerConfig {
    pub(crate) ip_address: Option<String>,
    pub(crate) port: Option<u16>,
}

#[derive(Deserialize, Debug, Default)]
pub(crate) struct ImageConfig {
    pub(crate) name: String,
    pub(crate) tag: String,
}

#[derive(Deserialize, Debug, Default)]
pub(crate) struct ContainerConfig {
    pub(crate) name: String,
    pub(crate) command: Vec<String>,
    pub(crate) ports: Vec<PortConfig>,
    pub(crate) mounts: Vec<MountConfig>,
}

#[derive(Deserialize, Debug, Default, Clone)]
pub(crate) struct PortConfig {
    pub(crate) host: u32,
    pub(crate) target: u32,
}

#[derive(Deserialize, Debug, Default)]
pub(crate) struct MountConfig {
    pub(crate) host: String,
    pub(crate) target: String,
}

#[derive(Deserialize, Debug, Default)]
pub struct BranchConfig {
    pub(crate) name: String,
    pub(crate) build_on_failure: bool,
}

#[derive(Deserialize, Debug, Default)]
pub struct HeartbeatConfig {
    pub(crate) sleep_time: i64,
    pub(crate) endpoint: String,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_example_config() {
        // This test makes sure the example config stays in line with the parsing code.
        let _config = DockerDeployConfig::from_file("config.toml.example");
    }
}
