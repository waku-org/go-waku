Receive and send messages using Waku Relay
===

Waku Relay is a gossip protocol that enables you to send and receive messages. You can find Waku Relay’s specifications on [Vac RFC](https://rfc.vac.dev/spec/11/).

## Create a waku instance
```go
package main

import (
	"context"
    "fmt"

	"github.com/waku-org/go-waku/waku/v2/node"
	"github.com/waku-org/go-waku/waku/v2/protocol/relay"
)

...
wakuNode, err := node.New(node.WithWakuRelay())
if err != nil {
    fmt.Println(err)
    return
}

if err := wakuNode.Start(context.Background()); err != nil {
    fmt.Println(err)
    return
}
...

```

### Options
One of these options must be specified when instantiating a node supporting the waku relay protocol

- `WithWakuRelay(opts ...pubsub.Option)` - enables the waku relay protocol and receives an optional list of pubsub options to tune or configure the gossipsub parameters. Supported options can be seen [here](https://pkg.go.dev/github.com/libp2p/go-libp2p-pubsub#Option). The recommended [parameter configuration](https://rfc.vac.dev/spec/29/) is used by default.
- `WithWakuRelayAndMinPeers(minRelayPeersToPublish int, opts ...pubsub.Option)` - enables the waku relay protocol, specifying the minimum number of peers a topic should have to send a message. It also receives an optional list of pubsub [options](https://pkg.go.dev/github.com/libp2p/go-libp2p-pubsub#Option)



## Receiving messages
```go
...
sub, err := wakuNode.Relay().Subscribe(context.Background())
if err != nil {
    fmt.Println(err)
    return
}

for value := range sub.C {
    fmt.Println("Received msg:", string(value.Message().Payload))
}
...
```
To receive messages sent via the relay protocol, you need to subscribe to a pubsub topic. This can be done via any of these functions:
- `wakuNode.Relay().Subscribe(ctx)` - subscribes to the default waku pubsub topic `/waku/2/default-waku/proto`
- `wakuNode.Relay().SubscribeToTopic(ctx, topic)` - subscribes to a custom pubsub topic

These functions return a `Subscription` struct containing a channel on which messages will be received. To stop receiving messages in this channel `sub.Unsubscribe()` can be executed which will close the channel (without unsubscribing from the pubsub topic)

> Pubsub topics should follow the [recommended usage](https://rfc.vac.dev/spec/23/) structure. For this purpose, the `NewPubsubTopic` helper function was created:
```go
import 	"github.com/waku-org/go-waku/waku/v2/protocol"

topic := protocol.NewPubsubTopic("the_topic_name", "the_encoding")
/*
fmt.Println(topic.String())  // => `/waku/2/the_topic_name/the_encoding`
*/
```



## Sending messages

```go
import (
    "github.com/waku-org/go-waku/waku/v2/protocol/pb"
    "github.com/waku-org/go-waku/waku/v2/utils"
)

...
msg := &pb.WakuMessage{
    Payload:      []byte("Hello World"),
    Version:      0,
    ContentTopic: protocol.NewContentTopic("basic2", 1, "test", "proto").String(),
    Timestamp:    utils.GetUnixEpoch(),
}

msgId, err = wakuNode.Relay().Publish(context.Background(), msg)
if err != nil {
    log.Error("Error sending a message: ", err)
}
```

To send a message, it needs to be wrapped into a [`WakuMessage`](https://rfc.vac.dev/spec/14/) protobuffer. The payload of the message is not limited to strings. Any kind of data that can be serialized
into a `[]byte` can be sent as long as it does not exceed the maximum length a message can have (~1MB)

The following functions can be used to publish a message:
- `wakuNode.Relay().Publish(ctx, msg)` - to send a message to the default waku pubsub topic
- `wakuNode.Relay().PublishToTopic(ctx, msg, topic)` - to send a message to a custom pubsub topic

Both of these functions will return a message id on success, or an error if the message could not be published.

> If `WithWakuRelayAndMinPeers` was used during the instantiation of the wakuNode, it should be possible to verify if there's enough peers for publishing to a topic with `wakuNode.Relay().EnoughPeersToPublish()` and `wakuNode.Relay().EnoughPeersToPublishToTopic(topic)`


## Stop receiving messages
```go
...
err := wakuNode.Relay().Unsubscribe(context.Background(), theTopic)
if err != nil {
    fmt.Println(err)
    return
}
...
```
> To stop receiving messages from the default topic, use `relay.DefaultWakuTopic` as the topic parameter

