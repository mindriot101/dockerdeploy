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
