use serde::Deserialize;
use tokio::sync::mpsc::{unbounded_channel, UnboundedReceiver};
use warp::Filter;

#[derive(Debug, Clone, Deserialize)]
struct Message {
    name: String,
}

// enum Message {
// Trigger,
// Webhook,
// Poll,
// }

struct Controller {
    rx: UnboundedReceiver<Message>,
}

impl Controller {
    async fn event_loop(&mut self) {
        while let Some(msg) = self.rx.recv().await {
            println!("{:?}", msg);
        }
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
            .and(json_body())
            .and(with_inbox(tx))
            .and_then(handlers::handle_trigger)
    }

    /// POST /api/webhook
    pub(crate) fn webhook(
        tx: UnboundedSender<Message>,
    ) -> impl Filter<Extract = impl warp::Reply, Error = warp::Rejection> + Clone {
        warp::path!("webhook")
            .and(warp::post())
            .and(json_body())
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
        // When accepting a body, we want a JSON body
        // (and to reject huge payloads)...
        warp::body::json()
    }
}

mod handlers {
    use crate::Message;
    use std::convert::Infallible;
    use tokio::sync::mpsc::UnboundedSender;
    use warp::http::StatusCode;

    pub(crate) async fn handle_trigger(
        msg: Message,
        _tx: UnboundedSender<Message>,
    ) -> Result<impl warp::Reply, Infallible> {
        eprintln!("message: {:?}", msg);
        Ok(StatusCode::NO_CONTENT)
    }

    pub(crate) async fn handle_webhook(
        msg: Message,
        _tx: UnboundedSender<Message>,
    ) -> Result<impl warp::Reply, Infallible> {
        eprintln!("message: {:?}", msg);
        Ok(StatusCode::NO_CONTENT)
    }
}

#[tokio::main]
async fn main() {
    let (tx, rx) = unbounded_channel();

    let mut controller = Controller { rx };

    tokio::spawn(async move {
        controller.event_loop().await;
    });

    let api = routes::build(tx);
    let routes = api.with(warp::log("dockerdeploy"));

    //     let hello = warp::path!("hello" / String).map(|name| async move {
    //         tx.send(()).await.unwrap();

    //         format!("Hello {}", name)
    //     });

    warp::serve(routes).run(([127, 0, 0, 1], 8080)).await;
}
