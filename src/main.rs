use anyhow::{Context, Result};
use bollard::Docker;
use notify::{RecommendedWatcher, RecursiveMode, Watcher};
use serde::Deserialize;
use std::path::PathBuf;
use structopt::StructOpt;
use tokio::sync::mpsc::{unbounded_channel, UnboundedReceiver, UnboundedSender};
use warp::Filter;

mod config;
mod dockerclient;
mod gitlab;
mod handlers;
mod routes;

use dockerclient::DockerApi;

#[derive(Debug, Clone, Deserialize)]
enum Message {
    Poll,
    Trigger,
    Reload(notify::event::Event),
}

struct Controller<D> {
    tx: UnboundedSender<Message>,
    rx: UnboundedReceiver<Message>,
    docker: D,
    cfg: config::DockerDeployConfig,
    cfg_file: PathBuf,
}

impl<D: DockerApi> Controller<D> {
    fn new(
        docker: D,
        cfg_file: PathBuf,
        tx: UnboundedSender<Message>,
        rx: UnboundedReceiver<Message>,
    ) -> Result<Self> {
        let config =
            config::DockerDeployConfig::from_file(&cfg_file).context("reading config file")?;
        log::debug!("got config {:?}", config);

        Ok(Controller {
            tx,
            rx,
            docker,
            cfg: config,
            cfg_file,
        })
    }

    async fn event_loop(&mut self) {
        while let Some(msg) = self.rx.recv().await {
            match msg {
                Message::Trigger => match self.trigger_refresh().await {
                    Ok(_) => {}
                    Err(e) => log::warn!("error in handler: {:?}", e),
                },
                Message::Poll => {
                    log::debug!("checking on container");

                    match self
                        .docker
                        .is_container_running(&self.cfg.container.name)
                        .await
                    {
                        Ok(r) => {
                            if !r {
                                log::info!("configured container not running, starting");
                                // Trigger a refresh
                                self.tx
                                    .send(Message::Trigger)
                                    .expect("sending trigger request");
                            } else {
                                log::info!(
                                    "found configured container `{}`",
                                    self.cfg.container.name
                                )
                            }
                        }
                        Err(e) => match e.downcast_ref::<bollard::errors::Error>() {
                            Some(e) => match e.kind() {
                                bollard::errors::ErrorKind::DockerResponseNotFoundError {
                                    ..
                                } => {
                                    log::info!("configured container not running, starting");
                                    // Trigger a refresh
                                    self.tx
                                        .send(Message::Trigger)
                                        .expect("sending trigger request");
                                }
                                _ => log::warn!(
                                    "error inspecting container {}: {:?}",
                                    self.cfg.container.name,
                                    e
                                ),
                            },
                            None => log::warn!(
                                "error inspecting container {}: {:?}",
                                self.cfg.container.name,
                                e
                            ),
                        },
                    }
                }
                Message::Reload(event) => {
                    use notify::event::EventKind;

                    log::trace!("reload event: {:?}", event);
                    match event.kind {
                        EventKind::Modify(_) => {
                            log::info!("reloading config");
                            let new_config = config::DockerDeployConfig::from_file(&self.cfg_file)
                                .expect("reading config file");
                            self.cfg = new_config;
                            log::info!("config reloaded: {:?}", self.cfg);
                        }
                        _ => {}
                    }
                }
            }
        }
    }

    async fn trigger_refresh(&mut self) -> Result<()> {
        self.pull_image().await?;
        self.stop_running_contianer().await?;
        self.run_container().await?;
        Ok(())
    }

    async fn pull_image(&mut self) -> Result<()> {
        use dockerclient::CreateImageOptions;

        log::info!("pulling image");

        let options = CreateImageOptions {
            from_image: self.cfg.image.name.as_str(),
            tag: self.cfg.image.tag.as_str(),
        };

        self.docker.create_image(options).await
    }

    async fn stop_running_contianer(&mut self) -> Result<()> {
        log::info!("stopping running container");

        match self.docker.remove_container(&self.cfg.container.name).await {
            Ok(_) => {}
            Err(e) => match e.downcast_ref::<bollard::errors::Error>() {
                Some(e) => match e.kind() {
                    bollard::errors::ErrorKind::DockerResponseNotFoundError { .. } => {
                        log::debug!("running container not found")
                    }
                    _ => anyhow::bail!("bad"),
                },
                None => anyhow::bail!("bad"),
            },
        }

        Ok(())
    }

    async fn run_container(&mut self) -> Result<()> {
        log::info!("running new container");

        let image = format!("{}:{}", self.cfg.image.name, self.cfg.image.tag);
        let cmd = self
            .cfg
            .container
            .command
            .iter()
            .map(|s| s.as_ref())
            .collect();
        self.docker
            .run_container(crate::dockerclient::RunContainerOptions {
                name: &self.cfg.container.name,
                image: &image,
                cmd,
            })
            .await?;

        Ok(())
    }
}

#[derive(StructOpt, Debug)]
#[structopt(name = "dockerdeploy", author = "Simon Walker")]
struct Opts {
    #[structopt(short, long, help = "Config file to parse", parse(from_os_str))]
    config: PathBuf,
}

#[tokio::main]
async fn main() {
    env_logger::init();

    let opts = Opts::from_args();
    log::trace!("command line options: {:?}", opts);

    let (tx, rx) = unbounded_channel();

    let docker = Docker::connect_with_local_defaults().expect("connecting to docker");
    let mut controller =
        Controller::new(docker, opts.config.clone(), tx.clone(), rx).expect("creating controller");

    let watcher_tx = tx.clone();
    let mut watcher: RecommendedWatcher =
        Watcher::new_immediate(move |res: notify::Result<notify::event::Event>| match res {
            Ok(event) => {
                watcher_tx
                    .send(Message::Reload(event.clone()))
                    .expect("reloading config");
            }
            Err(e) => eprintln!("error: {:?}", e),
        })
        .expect("creating watcher");

    watcher
        .watch(&opts.config, RecursiveMode::NonRecursive)
        .expect("failed to start watcher");

    // Start the poll loop
    let poll_tx = tx.clone();
    tokio::spawn(async move {
        log::info!("starting poll loop");
        loop {
            log::debug!("sending poll message");
            poll_tx.send(Message::Poll).expect("sending poll message");
            log::debug!("poll loop sleeping for 10 seconds");
            tokio::time::delay_for(std::time::Duration::from_secs(10)).await;
        }
    });

    tokio::spawn(async move {
        controller.event_loop().await;
    });

    let api = routes::build(tx);
    let routes = api.with(warp::log("dockerdeploy"));

    warp::serve(routes).run(([127, 0, 0, 1], 8080)).await;
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::dockerclient::{
        CreateContainerResults, CreateImageOptions, DockerApi, RunContainerOptions,
    };
    use anyhow::Result;
    use async_trait::async_trait;

    struct MockDocker;

    #[async_trait]
    impl DockerApi for MockDocker {
        async fn is_container_running(&self, container_name: &str) -> Result<bool> {
            todo!()
        }

        async fn remove_container(&self, container_name: &str) -> Result<()> {
            todo!()
        }

        async fn run_container<'a>(
            &'a self,
            options: RunContainerOptions<'a>,
        ) -> Result<CreateContainerResults> {
            todo!()
        }

        async fn create_image<'a>(&'a self, options: CreateImageOptions<'a>) -> Result<()> {
            todo!()
        }
    }

    #[tokio::test]
    async fn test_creating_custom_controller() {
        let docker = MockDocker {};
        let (tx, rx) = unbounded_channel();
        let config = PathBuf::from("config.toml.example");
        let controller = Controller::new(docker, config, tx, rx).unwrap();
        assert!(true);
    }
}
