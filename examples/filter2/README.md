# Using the `filter2` application

## Background

The `filter2` application is a basic example app that demonstrates how to subscribe to and publish messages using waku2-filter

## Preparation
```
make
```

## Basic application usage

To start the `filter2` application run the following from the project directory

```
./build/filter2
```
The app will run 2 nodes ("full" node and "light" node), with light node subscribing to full node in order to receive filtered messages.


## Flow description

1. Light node submits a FilterRequest through WakuNode.SubscribeFilter. This request is submitted to a particular peer.
Filter is stored in WakuNode.filters map. That's it.
DONE
2. Full node: we read incoming messages in WakuFilter.onRequest(). It is set as a stream handler on wakunode.Host for WakuFilterProtocolId.
3. In WakuFilter.onRequest():
  3.1. We check whether it's a MessagePush or FilterRequest.
  3.2. If it's a MessagePush, then we're on a light node. Invoke pushHandler coming from WakuNode.mountFilter()
  3.3. If it's a FilterRequest, add a subscriber.
4. WakuNode.Subscribe has a message loop extracting WakuMessages from a wakurelay.Subscription object.
It denotes a pubsub topic subscription.
All envelopes are then submitted to node.broadcaster.


## Nim code flow
1. Light node: WakuFilter.subscribe(). Find a peer, wrileLP(FilterRequest). Store requestId in WakuNode.filters along with a ContentFilterHandler proc.
2. Full node: WakuFilter inherits LPProtocol. LPProtocol.handler invokes readLP() to read FilterRPC messages
3. this handler function has a signature (conn: Connection, proto: string).
  3.1. it checks whether a MessagePush or FilterRequest is received.
  3.2. (light node) if it's a MessagePush, then we're on a light node. Invoke pushHandler of MessagePushHandler type. This pushHandler comes from WakuNode.mountFilter(). It iterates through all registered WakuNode.filters (stored in step 1) and invokes their ContentFilterHandler proc.
  3.3. (full node) if it's a FilterRequest, create a Subscriber and add to WakuFilter.subscribers seq
4. (full node) Each time a message is received through GossipSub in wakunode.subscribe.defaultHandler(), we iterate through subscriptions.
5. (full node) One of these subscriptions is a filter subscription added by WakuNode.mountFilter(), which in turn is returned from WakuFilter.subscription()
6. (full node) This subscription iterates through subscribers added by WakuFilter.handler() fn (subscribers being light nodes)
7. (full node) Once subscriber peer is found, a message is pushed directly to the peer (go to step 3.2)

