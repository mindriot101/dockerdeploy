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
) -> impl Filter<Extract = (UnboundedSender<Message>,), Error = std::convert::Infallible> + Clone {
    warp::any().map(move || tx.clone())
}

fn json_body() -> impl Filter<Extract = (Message,), Error = warp::Rejection> + Clone {
    warp::body::json()
}
