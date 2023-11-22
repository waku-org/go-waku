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

### Light Node
1. A light node is created with option WithWakuFilterLightNode.
2. Starting this node sets stream handler on wakunode.Host for WakuFilterProtocolId.
3. Light node submits a FilterSubscribeRequest through WakuFilterLightNode.Subscribe. This request is submitted to a particular peer.
Filter is stored in WakuFilterLightNode.subscriptions map. That's it.
4. Now we wait on WakuFilterLightNode.onRequest to process any further messages.
5. On receiving a message check and notify all subscribers on relevant channel (which is part of subscription object).
6. If a broadcaster is specified, 
  WakuNode.Subscribe has a message loop extracting WakuMessages from a wakurelay.Subscription object. It denotes a pubsub topic subscription. All envelopes are then submitted to node.broadcaster.
### Full Node
1. Full node is created with option WithWakuFilterFullNode.
2. We read incoming messages in WithWakuFilterFullNode.onRequest(). It is set as a stream handler on wakunode.Host for WakuFilterProtocolId.
3. In WakuFilter.onRequest
  * We check the type of FilterRequest and handle accordingly.
  * If it's a FilterRequest for subscribe, add a subscriber.
  * If it is a SubscriberPing request, check if subscriptions exists or not and respond accordingly.
  *  If it is an unsubscribe/unsubscribeAll request, check and remove relevant subscriptions.
