use anyhow::Result;
use bollard::Docker;
use notify::{RecommendedWatcher, RecursiveMode, Watcher};
use serde::Deserialize;
use std::path::PathBuf;
use structopt::StructOpt;
use tokio::stream::StreamExt;
use tokio::sync::mpsc::{unbounded_channel, UnboundedReceiver};
use warp::Filter;

mod config;
mod handlers;
mod routes;

#[derive(Debug, Clone, Deserialize)]
enum Message {
    Trigger,
    Reload(notify::event::Event),
}

struct Controller {
    rx: UnboundedReceiver<Message>,
    docker: Docker,
    cfg: config::DockerDeployConfig,
    cfg_file: PathBuf,
}

impl Controller {
    async fn event_loop(&mut self) {
        while let Some(msg) = self.rx.recv().await {
            match msg {
                Message::Trigger => match self.trigger_refresh().await {
                    Ok(_) => {}
                    Err(e) => log::warn!("error in handler: {:?}", e),
                },
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
        use bollard::image::CreateImageOptions;

        log::info!("pulling image");

        let options = Some(CreateImageOptions {
            from_image: self.cfg.image.name.as_str(),
            tag: self.cfg.image.tag.as_str(),
            ..Default::default()
        });

        let mut out_stream = self.docker.create_image(options, None, None);
        while let Some(msg) = out_stream.next().await {
            log::debug!("{:?}", msg);
        }
        Ok(())
    }

    async fn stop_running_contianer(&mut self) -> Result<()> {
        use bollard::container::RemoveContainerOptions;

        log::info!("stopping running container");

        let options = Some(RemoveContainerOptions {
            force: true,
            ..Default::default()
        });

        match self
            .docker
            .remove_container(&self.cfg.container.name, options)
            .await
        {
            Ok(_) => {}
            Err(e) => match e.kind() {
                bollard::errors::ErrorKind::DockerResponseNotFoundError { .. } => {
                    log::debug!("running container not found")
                }
                _ => return Err(e.into()),
            },
        }

        Ok(())
    }

    async fn run_container(&mut self) -> Result<()> {
        use bollard::container::{Config, CreateContainerOptions, StartContainerOptions};

        log::info!("running new container");

        let options = Some(CreateContainerOptions {
            name: self.cfg.container.name.clone(),
            ..Default::default()
        });

        let config = Config {
            image: Some(format!("{}:{}", self.cfg.image.name, self.cfg.image.tag)),
            cmd: Some(self.cfg.container.command.clone()),
            ..Default::default()
        };

        let res = self.docker.create_container(options, config).await?;
        for warning in res.warnings.unwrap_or(Vec::new()) {
            log::warn!("container create warning: {}", warning);
        }

        // Start the new container
        self.docker
            .start_container(&res.id, None::<StartContainerOptions<String>>)
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

    let config = config::DockerDeployConfig::from_file(&opts.config).expect("reading config file");
    log::debug!("got config {:#?}", config);

    let (tx, rx) = unbounded_channel();

    let docker = Docker::connect_with_local_defaults().expect("connecting to docker");

    let mut controller = Controller {
        rx,
        docker,
        cfg: config,
        cfg_file: opts.config.clone(),
    };

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
        .watch(opts.config, RecursiveMode::NonRecursive)
        .expect("failed to start watcher");

    tokio::spawn(async move {
        controller.event_loop().await;
    });

    let api = routes::build(tx);
    let routes = api.with(warp::log("dockerdeploy"));

    warp::serve(routes).run(([127, 0, 0, 1], 8080)).await;
}
