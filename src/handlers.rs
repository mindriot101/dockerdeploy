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
    header_key: Option<String>,
    event: Event,
    tx: UnboundedSender<Message>,
    validation_value: Option<String>,
) -> Result<impl warp::Reply, Infallible> {
    // Check that the incoming event is a gitlab one and that matches the pipeline event type
    log::debug!("got event {:?}", event);

    // Check the header key matches
    if let Some(val) = validation_value {
        match header_key {
            Some(key) => {
                // Check that the header key matches the expected value in the config file
                if val != key {
                    log::warn!("expected key does not match request");
                    return Ok(StatusCode::UNAUTHORIZED);
                }

                log::info!("expected key matches request, continuing");
                if let Event::Pipeline(pipeline) = event {
                    log::debug!("pipeline event configured to run new deploy");
                    if pipeline.should_rerun_pipeline() {
                        log::info!("webhook trigger accepted");
                        tx.send(Message::Trigger).unwrap();
                    } else {
                        log::info!("webhook trigger rejected");
                    }
                } else {
                    log::debug!("{:?} event _not_ configured to run new deploy", event);
                }

                Ok(StatusCode::NO_CONTENT)
            }
            None => {
                log::warn!("X-Gitlab-Token header key not given; validation is required");
                Ok(StatusCode::UNAUTHORIZED)
            }
        }
    } else {
        Ok(StatusCode::UNAUTHORIZED)
    }
}
