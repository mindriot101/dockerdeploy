use anyhow::Result;
use serde::Deserialize;
use tokio::sync::mpsc::{unbounded_channel, UnboundedReceiver};
use warp::Filter;

#[derive(Debug, Clone, Deserialize)]
enum Message {
    Trigger,
}

struct Controller {
    rx: UnboundedReceiver<Message>,
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
        log::info!("pulling image");
        Ok(())
    }

    async fn stop_running_contianer(&mut self) -> Result<()> {
        log::info!("stopping running container");
        Ok(())
    }

    async fn run_container(&mut self) -> Result<()> {
        log::info!("running new container");
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

    let mut controller = Controller { rx };

    tokio::spawn(async move {
        controller.event_loop().await;
    });

    let api = routes::build(tx);
    let routes = api.with(warp::log("dockerdeploy"));

    warp::serve(routes).run(([127, 0, 0, 1], 8080)).await;
}
