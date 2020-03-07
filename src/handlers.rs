use crate::gitlab::Event;
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
    event: Event,
    tx: UnboundedSender<Message>,
) -> Result<impl warp::Reply, Infallible> {
    // Check that the incoming event is a gitlab one and that matches the pipeline event type
    log::debug!("got event {:?}", event);

    let Event::Pipeline(pipeline) = event;
    if pipeline.should_rerun_pipeline() {
        log::info!("webhook trigger accepted");
        tx.send(Message::Trigger).unwrap();
    } else {
        log::info!("webhook trigger rejected");
    }

    Ok(StatusCode::NO_CONTENT)
}
