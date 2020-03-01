use tokio::sync::mpsc::{unbounded_channel, UnboundedReceiver};
use warp::Filter;

struct Controller {
    rx: UnboundedReceiver<()>,
}

impl Controller {
    async fn event_loop(&mut self) {
        while let Some(msg) = self.rx.recv().await {
            println!("{:?}", msg);
        }
    }
}

#[tokio::main]
async fn main() {
    let (tx, rx) = unbounded_channel();

    let mut controller = Controller { rx };

    tokio::spawn(async move {
        controller.event_loop().await;
    });

    let hello = warp::path!("hello" / String).map(|name| {
        let tx = tx.clone();
        async move {
            tx.send(()).await.unwrap();
        };

        format!("Hello {}", name)
    });

    warp::serve(hello).run(([127, 0, 0, 1], 8080)).await;
}
