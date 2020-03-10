use crate::gitlab::Event;
use crate::handlers;
use crate::Message;
use tokio::sync::mpsc::UnboundedSender;
use warp::filters::header::optional;
use warp::Filter;

pub(crate) fn build(
    tx: UnboundedSender<Message>,
    validation_key: Option<String>,
) -> impl Filter<Extract = impl warp::Reply, Error = warp::Rejection> + Clone {
    trigger(tx.clone()).or(webhook(tx, validation_key))
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
    validation_key: Option<String>,
) -> impl Filter<Extract = impl warp::Reply, Error = warp::Rejection> + Clone {
    let key = warp::any().map(move || validation_key.clone());

    warp::path!("webhook")
        .and(warp::post())
        .and(optional::<String>("X-Gitlab-Token"))
        .and(json_body())
        .and(with_inbox(tx))
        .and(key)
        .and_then(handlers::handle_webhook)
}

fn with_inbox(
    tx: UnboundedSender<Message>,
) -> impl Filter<Extract = (UnboundedSender<Message>,), Error = std::convert::Infallible> + Clone {
    warp::any().map(move || tx.clone())
}

fn json_body() -> impl Filter<Extract = (Event,), Error = warp::Rejection> + Clone {
    warp::body::json()
}
