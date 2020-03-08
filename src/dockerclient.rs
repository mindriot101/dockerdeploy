use anyhow::Result;
use async_trait::async_trait;
use bollard::Docker;
use tokio::stream::StreamExt;

pub(crate) struct RunContainerOptions<'a> {
    pub(crate) name: &'a str,
    pub(crate) image: &'a str,
    pub(crate) cmd: Vec<&'a str>,
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
        let options = Some(InspectContainerOptions {
            size: false,
            ..Default::default()
        });

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
        use bollard::container::{Config, CreateContainerOptions, StartContainerOptions};

        let c_options = Some(CreateContainerOptions {
            name: options.name,
            ..Default::default()
        });

        let cmd = options.cmd;
        let config = Config {
            image: Some(options.image),
            cmd: Some(cmd),
            ..Default::default()
        };

        let res = Docker::create_container(self, c_options, config).await?;

        // Start the new container
        Docker::start_container(self, &res.id, None::<StartContainerOptions<String>>).await?;

        Ok(CreateContainerResults {
            warnings: res.warnings.unwrap_or_else(|| Vec::new()),
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
