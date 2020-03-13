use anyhow::Result;
use async_trait::async_trait;
use bollard::container::PortBinding;
use bollard::Docker;
use std::collections::HashMap;
use tokio::stream::StreamExt;

pub(crate) struct RunContainerOptions<'a> {
    pub(crate) name: &'a str,
    pub(crate) image: &'a str,
    pub(crate) cmd: Vec<&'a str>,
    pub(crate) ports: Vec<crate::config::PortConfig>,
}

trait PortConfigMap {
    fn to_bollard(&self) -> HashMap<String, Vec<PortBinding<String>>>;
}

impl PortConfigMap for Vec<crate::config::PortConfig> {
    fn to_bollard(&self) -> HashMap<String, Vec<PortBinding<String>>> {
        self.iter()
            .map(|config| {
                (
                    format!("{}/tcp", config.target),
                    vec![PortBinding {
                        host_port: format!("{}/tcp", config.host),
                        host_ip: format!("0.0.0.0"),
                    }],
                )
            })
            .collect()
    }
}

pub(crate) struct CreateImageOptions<'a> {
    pub(crate) from_image: &'a str,
    pub(crate) tag: &'a str,
}

pub(crate) struct CreateContainerResults {
    pub(crate) warnings: Vec<String>,
}

#[async_trait]
pub(crate) trait DockerApi {
    async fn is_container_running(&self, container_name: &str) -> Result<bool>;

    async fn remove_container(&self, container_name: &str) -> Result<()>;

    async fn run_container<'a>(
        &'a self,
        options: RunContainerOptions<'a>,
    ) -> Result<CreateContainerResults>;

    async fn create_image<'a>(&'a self, options: CreateImageOptions<'a>) -> Result<()>;
}

#[async_trait]
impl DockerApi for bollard::Docker {
    async fn is_container_running(&self, container_name: &str) -> Result<bool> {
        use bollard::container::InspectContainerOptions;
        let options = Some(InspectContainerOptions { size: false });

        match Docker::inspect_container(self, container_name, options).await {
            Ok(_) => Ok(true),
            Err(e) => match e.kind() {
                bollard::errors::ErrorKind::DockerResponseNotFoundError { .. } => {
                    log::info!("configured container not running, starting");
                    Ok(false)
                }
                _ => {
                    log::warn!("error inspecting container {}: {:?}", container_name, e);
                    Err(e.into())
                }
            },
        }
    }

    async fn remove_container(&self, container_name: &str) -> Result<()> {
        use bollard::container::RemoveContainerOptions;

        let options = Some(RemoveContainerOptions {
            force: true,
            ..Default::default()
        });

        let res = Docker::remove_container(self, container_name, options).await;

        match res {
            Ok(_) => Ok(()),
            Err(e) => match e.kind() {
                bollard::errors::ErrorKind::DockerResponseNotFoundError { .. } => {
                    log::debug!("running container not found");
                    Ok(())
                }
                _ => Err(e.into()),
            },
        }
    }

    async fn run_container<'a>(
        &'a self,
        options: RunContainerOptions<'a>,
    ) -> Result<CreateContainerResults> {
        use bollard::container::{
            Config, CreateContainerOptions, HostConfig, StartContainerOptions,
        };

        let c_options = Some(CreateContainerOptions { name: options.name });

        let host_config = Some(HostConfig {
            port_bindings: Some(options.ports.to_bollard()),
            ..Default::default()
        });

        let cmd = options.cmd.iter().map(|s| s.to_string()).collect();
        let config = Config {
            image: Some(options.image.to_string()),
            cmd: Some(cmd),
            host_config,
            ..Default::default()
        };

        let res = Docker::create_container(self, c_options, config).await?;

        // Start the new container
        Docker::start_container(self, &res.id, None::<StartContainerOptions<String>>).await?;

        Ok(CreateContainerResults {
            warnings: res.warnings.unwrap_or_else(Vec::new),
        })
    }

    async fn create_image<'a>(&'a self, options: CreateImageOptions<'a>) -> Result<()> {
        use bollard::image;

        let options = Some(image::CreateImageOptions {
            from_image: options.from_image,
            tag: options.tag,
            ..Default::default()
        });

        let mut out_stream = Docker::create_image(self, options, None, None);
        while let Some(msg) = out_stream.next().await {
            log::debug!("{:?}", msg);
        }
        Ok(())
    }
}
