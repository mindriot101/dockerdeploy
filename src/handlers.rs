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

    // Should we trigger a pipeline build?
    let ok = validation_value.map_or(true, |val| header_key.map_or(false, |key| val == key));

    if ok {
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
    } else {
        Ok(StatusCode::UNAUTHORIZED)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::gitlab::{Event, ObjectAttributes, Pipeline};
    use tokio::sync::mpsc::unbounded_channel;
    use warp::reply::Reply;

    #[tokio::test]
    async fn test_webhook_happy_path() {
        let header_key = Some("abc".to_string());
        let event = Event::Pipeline(Pipeline {
            builds: Vec::new(),
            object_attributes: ObjectAttributes {
                object_ref: "master".to_string(),
            },
        });
        // TODO: check response from channel
        let (tx, _rx) = unbounded_channel();
        let validation_value = Some("abc".to_string());

        let res = handle_webhook(header_key, event, tx, validation_value)
            .await
            .unwrap();

        let response = res.into_response();
        assert_eq!(response.status(), StatusCode::NO_CONTENT);
    }

    #[tokio::test]
    async fn test_webhook_key_doesnt_match() {
        let header_key = Some("abcd".to_string());
        let event = Event::Pipeline(Pipeline {
            builds: Vec::new(),
            object_attributes: ObjectAttributes {
                object_ref: "master".to_string(),
            },
        });
        // TODO: check response from channel
        let (tx, _rx) = unbounded_channel();
        let validation_value = Some("abc".to_string());

        let res = handle_webhook(header_key, event, tx, validation_value)
            .await
            .unwrap();

        let response = res.into_response();
        assert_eq!(response.status(), StatusCode::UNAUTHORIZED);
    }

    #[tokio::test]
    async fn test_webhook_key_not_given() {
        let header_key = None;
        let event = Event::Pipeline(Pipeline {
            builds: Vec::new(),
            object_attributes: ObjectAttributes {
                object_ref: "master".to_string(),
            },
        });
        // TODO: check response from channel
        let (tx, _rx) = unbounded_channel();
        let validation_value = Some("abc".to_string());

        let res = handle_webhook(header_key, event, tx, validation_value)
            .await
            .unwrap();

        let response = res.into_response();
        assert_eq!(response.status(), StatusCode::UNAUTHORIZED);
    }

    #[tokio::test]
    async fn test_webhook_no_key_given() {
        let header_key = Some("abc".to_string());
        let event = Event::Pipeline(Pipeline {
            builds: Vec::new(),
            object_attributes: ObjectAttributes {
                object_ref: "master".to_string(),
            },
        });
        // TODO: check response from channel
        let (tx, _rx) = unbounded_channel();
        let validation_value = None;

        let res = handle_webhook(header_key, event, tx, validation_value)
            .await
            .unwrap();

        let response = res.into_response();
        assert_eq!(response.status(), StatusCode::NO_CONTENT);
    }

    #[tokio::test]
    async fn test_webhook_no_keys_given() {
        let header_key = None;
        let event = Event::Pipeline(Pipeline {
            builds: Vec::new(),
            object_attributes: ObjectAttributes {
                object_ref: "master".to_string(),
            },
        });
        // TODO: check response from channel
        let (tx, _rx) = unbounded_channel();
        let validation_value = None;

        let res = handle_webhook(header_key, event, tx, validation_value)
            .await
            .unwrap();

        let response = res.into_response();
        assert_eq!(response.status(), StatusCode::NO_CONTENT);
    }
}
