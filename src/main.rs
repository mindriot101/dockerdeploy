use anyhow::Result;
use bollard::Docker;
use serde::Deserialize;
use tokio::stream::StreamExt;
use tokio::sync::mpsc::{unbounded_channel, UnboundedReceiver};
use warp::Filter;

static IMAGE_NAME: &'static str = "ubuntu";
static CONTAINER_NAME: &'static str = "foobar";

#[derive(Debug, Clone, Deserialize)]
enum Message {
    Trigger,
}

struct Controller {
    rx: UnboundedReceiver<Message>,
    docker: Docker,
}

impl Controller {
    async fn event_loop(&mut self) {
        while let Some(msg) = self.rx.recv().await {
            match msg {
                Message::Trigger => match self.trigger_refresh().await {
                    Ok(_) => {}
                    Err(e) => log::warn!("error in handler: {:?}", e),
                },
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
            from_image: IMAGE_NAME,
            tag: "latest",
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

        match self.docker.remove_container(CONTAINER_NAME, options).await {
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
            name: CONTAINER_NAME,
            ..Default::default()
        });

        let config = Config {
            image: Some(IMAGE_NAME),
            cmd: Some(vec!["sleep", "86400"]),
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

mod routes {
    use crate::handlers;
    use crate::Message;
    use tokio::sync::mpsc::UnboundedSender;
    use warp::Filter;

    pub(crate) fn build(
        tx: UnboundedSender<Message>,
    ) -> impl Filter<Extract = impl warp::Reply, Error = warp::Rejection> + Clone {
        trigger(tx.clone()).or(webhook(tx.clone()))
    }

    /// POST /api/trigger
    pub(crate) fn trigger(
        tx: UnboundedSender<Message>,
    ) -> impl Filter<Extract = impl warp::Reply, Error = warp::Rejection> + Clone {
        warp::path!("trigger")
            .and(warp::post())
            .and(with_inbox(tx))
            .and_then(handlers::handle_trigger)
    }

    /// POST /api/webhook
    pub(crate) fn webhook(
        tx: UnboundedSender<Message>,
    ) -> impl Filter<Extract = impl warp::Reply, Error = warp::Rejection> + Clone {
        warp::path!("webhook")
            .and(warp::post())
            // .and(json_body())
            .and(with_inbox(tx))
            .and_then(handlers::handle_webhook)
    }

    fn with_inbox(
        tx: UnboundedSender<Message>,
    ) -> impl Filter<Extract = (UnboundedSender<Message>,), Error = std::convert::Infallible> + Clone
    {
        warp::any().map(move || tx.clone())
    }

    fn json_body() -> impl Filter<Extract = (Message,), Error = warp::Rejection> + Clone {
        warp::body::json()
    }
}

mod handlers {
    use crate::Message;
    use std::convert::Infallible;
    use tokio::sync::mpsc::UnboundedSender;
    use warp::http::StatusCode;

    pub(crate) async fn handle_trigger(
        tx: UnboundedSender<Message>,
    ) -> Result<impl warp::Reply, Infallible> {
        tx.send(Message::Trigger).unwrap();

        Ok(StatusCode::NO_CONTENT)
    }

    pub(crate) async fn handle_webhook(
        _tx: UnboundedSender<Message>,
    ) -> Result<impl warp::Reply, Infallible> {
        Ok(StatusCode::NO_CONTENT)
    }
}

#[tokio::main]
async fn main() {
    env_logger::init();

    let (tx, rx) = unbounded_channel();

    let docker = Docker::connect_with_local_defaults().expect("connecting to docker");

    let mut controller = Controller { rx, docker };

    tokio::spawn(async move {
        controller.event_loop().await;
    });

    let api = routes::build(tx);
    let routes = api.with(warp::log("dockerdeploy"));

    warp::serve(routes).run(([127, 0, 0, 1], 8080)).await;
}
