This specification describes the API for consuming go-waku when built as a dynamic or static library


# libgowaku.h


# Introduction

Native applications that wish to integrate Waku may not be able to use nwaku and its JSON RPC API due to constraints
on packaging, performance or executables.

An alternative is to link existing Waku implementation as a static or dynamic library in their application.

This specification describes the C API that SHOULD be implemented by native Waku library and that SHOULD be used to
consume them.

# Design requirements

The API should be generic enough, so:

- it can be implemented by both nwaku and go-waku C-Bindings,
- it can be consumed from a variety of languages such as C#, Kotlin, Swift, Rust, C++, etc.

The selected format to pass data to and from the API is `JSON`.

It has been selected due to its widespread usage and easiness of use. Other alternatives MAY replace it in the future (C
structure, protobuf) if it brings limitations that need to be lifted.

# The API

## General

### `JsonResponse` type

All the API functions return a `JsonResponse` unless specified otherwise.
`JsonResponse` is a `char *` whose format depends on whether the function was executed successfully or not.

On failure:

```ts
{
    error: string;
}
```

For example: 

```json
{
  "error": "the error message"
}
```

On success:

```ts
{
    result: any
}
```

The type of the `result` object depends on the function it was returned by. 

### `JsonMessage` type

A Waku Message in JSON Format:

```ts
{
    payload: string;
    contentTopic: string;
    version: number;
    timestamp: number;
}
```

Fields:

- `payload`: base64 encoded payload, [`waku_utils_base64_encode`](#extern-char-waku_utils_base64_encodechar-data) can be used for this.
- `contentTopic`: The content topic to be set on the message.
- `version`: The Waku Message version number.
- `timestamp`: Unix timestamp in nanoseconds.

### `DecodedPayload` type

A payload once decoded, used when a received Waku Message is encrypted:

```ts
interface DecodedPayload {
    pubkey?: string;
    signature?: string;
    data: string;
    padding: string;
  }
```

Fields:

- `pubkey`: Public key that signed the message (optional), hex encoded with `0x` prefix,
- `signature`: Message signature (optional), hex encoded with `0x` prefix,
- `data`: Decrypted message payload base64 encoded,
- `padding`: Padding base64 encoded.

### `FilterSubscription` type

The criteria to create subscription to a light node in JSON Format:

```ts
{
    contentFilters: ContentFilter[];
    pubsubTopic: string?;
}
```

Fields:

- `contentFilters`: Array of [`ContentFilter`](#contentfilter-type) being subscribed to / unsubscribed from.
- `topic`: Optional pubsub topic.


### `ContentFilter` type

```ts
{
    contentTopic: string;
}
```

Fields:

- `contentTopic`: The content topic of a Waku message.

### `StoreQuery` type

Criteria used to retrieve historical messages

```ts
interface StoreQuery {
    pubsubTopic?: string;
    contentFilters?: ContentFilter[];
    startTime?: number;
    endTime?: number;
    pagingOptions?: PagingOptions
  }
```

Fields:

- `pubsubTopic`: The pubsub topic on which messages are published.
- `contentFilters`: Array of [`ContentFilter`](#contentfilter-type) to query for historical messages,
- `startTime`: The inclusive lower bound on the timestamp of queried messages. This field holds the Unix epoch time in nanoseconds.
- `endTime`: The inclusive upper bound on the timestamp of queried messages. This field holds the Unix epoch time in nanoseconds.
- `pagingOptions`: Paging information in [`PagingOptions`](#pagingoptions-type) format.

### `StoreResponse` type

The response received after doing a query to a store node:

```ts
interface StoreResponse {
    messages: JsonMessage[];
    pagingOptions?: PagingOptions;
  }
```
Fields:

- `messages`: Array of retrieved historical messages in [`JsonMessage`](#jsonmessage-type) format.
- `pagingOption`: Paging information in [`PagingOptions`](#pagingoptions-type) format from which to resume further historical queries

### `PagingOptions` type

```ts
interface PagingOptions {
    pageSize: number;
    cursor?: Index;
    forward: bool;
  }
```
Fields:

- `pageSize`: Number of messages to retrieve per page.
- `cursor`: Message Index from which to perform pagination. If not included and forward is set to true, paging will be performed from the beginning of the list. If not included and forward is set to false, paging will be performed from the end of the list.
- `forward`:  `true` if paging forward, `false` if paging backward

### `Index` type

```ts
interface Index {
    digest: string;
    receiverTime: number;
    senderTime: number;
    pubsubTopic: string;
  }
```

Fields:

- `digest`: Hash of the message at this [`Index`](#index-type).
- `receiverTime`: UNIX timestamp in nanoseconds at which the message at this [`Index`](#index-type) was received.
- `senderTime`: UNIX timestamp in nanoseconds at which the message is generated by its sender.
- `pubsubTopic`: The pubsub topic of the message at this [`Index`](#index-type).

## Events

Asynchronous events require a callback to be registered.
An example of an asynchronous event that might be emitted is receiving a message.
When an event is emitted, this callback will be triggered receiving a JSON string of type `JsonSignal`.

### `JsonSignal` type

```ts
{
    type: string;
    event: any;
}
```

Fields:

- `type`: Type of signal being emitted. Currently, only `message` is available.
- `event`: Format depends on the type of signal.

For example:

```json
{
  "type": "message",
  "event": {
    "subscriptionId": 1,
    "pubsubTopic": "/waku/2/default-waku/proto",
    "messageId": "0x6496491e40dbe0b6c3a2198c2426b16301688a2daebc4f57ad7706115eac3ad1",
    "wakuMessage": {
      "payload": "TODO",
      "contentTopic": "/my-app/1/notification/proto",
      "version": 1,
      "timestamp": 1647826358000000000
    }
  }
}
```

| `type`    | `event` Type       |
|:----------|--------------------|
| `message` | `JsonMessageEvent` | 

### `JsonMessageEvent` type

Type of `event` field for a `message` event:

```ts
{
    pubsubTopic: string;
    messageId: string;
    wakuMessage: JsonMessage;
}
```

- `pubsubTopic`: The pubsub topic on which the message was received.
- `messageId`: The message id.
- `wakuMessage`: The message in [`JsonMessage`](#jsonmessage-type) format.

### `extern void waku_set_event_callback(void* cb)`

Register callback to act as event handler and receive application signals,
which are used to react to asynchronous events in Waku.

**Parameters**

1. `void* cb`: callback that will be executed when an async event is emitted.
  The function signature for the callback should be `void myCallback(char* jsonSignal)`

## Node management

### `JsonConfig` type

Type holding a node configuration:

```ts
interface JsonSignal {
    host?: string;
    port?: number;
    advertiseAddr?: string;
    nodeKey?: string;
    keepAliveInterval?: number;
    relay?: boolean;
    minPeersToPublish?: number
    filter?: boolean;
}
```

Fields: 

All fields are optional.
If a key is `undefined`, or `null`, a default value will be set.

- `host`: Listening IP address.
  Default `0.0.0.0`.
- `port`: Libp2p TCP listening port.
  Default `60000`. 
  Use `0` for random.
- `advertiseAddr`: External address to advertise to other nodes.
  Can be ip4, ip6 or dns4, dns6.
  If `null`, the multiaddress(es) generated from the ip and port specified in the config (or default ones) will be used.
  Default: `null`.
- `nodeKey`: Secp256k1 private key in Hex format (`0x123...abc`).
  Default random.
- `keepAliveInterval`: Interval in seconds for pinging peers to keep the connection alive.
  Default `20`.
- `relay`: Enable relay protocol.
  Default `true`.
- `minPeersToPublish`: The minimum number of peers required on a topic to allow broadcasting a message.
  Default `0`.
- `filter`: Enable filter protocol.
  Default `false`.

For example:
```json
{
  "host": "0.0.0.0",
  "port": 60000,
  "advertiseAddr": "1.2.3.4",
  "nodeKey": "0x123...567",
  "keepAliveInterval": 20,
  "relay": true,
  "minPeersToPublish": 0
}
```

### `extern char* waku_new(char* jsonConfig)`

Instantiates a Waku node.

**Parameters**

1. `char* jsonConfig`: JSON string containing the options used to initialize a go-waku node.
   Type [`JsonConfig`](#jsonconfig-type).
   It can be `NULL` to use defaults.

**Returns**

A [`JsonResponse`](#jsonresponse-type).
If the execution is successful, the `result` field is set to `true`.

For example:

```json
{
  "result": true
}
```

### `extern char* waku_start()`

Start a Waku node mounting all the protocols that were enabled during the Waku node instantiation.

**Returns**

A [`JsonResponse`](#jsonresponse-type).
If the execution is successful, the `result` field is set to `true`.

For example:

```json
{
  "result": true
}
```

### `extern char* waku_stop()`

Stops a Waku node.

**Returns**

A [`JsonResponse`](#jsonresponse-type).
If the execution is successful, the `result` field is set to `true`.

For example:

```json
{
  "result": true
}
```

### `extern char* waku_peerid()`

Get the peer ID of the waku node.

**Returns**

A [`JsonResponse`](#jsonresponse-type).
If the execution is successful, the `result` field contains the peer ID as a `string` (base58 encoded).

For example:

```json
{
  "result": "QmWjHKUrXDHPCwoWXpUZ77E8o6UbAoTTZwf1AD1tDC4KNP"
}
```

### `extern char* waku_listen_addresses()`

Get the multiaddresses the Waku node is listening to.

**Returns**

A [`JsonResponse`](#jsonresponse-type).
If the execution is successful, the `result` field contains an array of multiaddresses.
The multiaddresses are `string`s.

For example:

```json
{
  "result": [
    "/ip4/127.0.0.1/tcp/30303",
    "/ip4/1.2.3.4/tcp/30303",
    "/dns4/waku.node.example/tcp/8000/wss"
  ]
}
```

## Connecting to peers

### `extern char* waku_add_peer(char* address, char* protocolId)`

Add a node multiaddress and protocol to the waku node's peerstore.

**Parameters**

1. `char* address`: A multiaddress (with peer id) to reach the peer being added.
2. `char* protocolId`: A protocol we expect the peer to support.

**Returns**

A [`JsonResponse`](#jsonresponse-type).
If the execution is successful, the `result` field contains the peer ID as a base58 `string` of the peer that was added.

For example:

```json
{
  "result": "QmWjHKUrXDHPCwoWXpUZ77E8o6UbAoTTZwf1AD1tDC4KNP"
}
```

### `extern char* waku_connect_peer(char* address, int timeoutMs)`

Dial peer using a multiaddress.

**Parameters**

1. `char* address`: A multiaddress to reach the peer being dialed.
2. `int timeoutMs`: Timeout value in milliseconds to execute the call.
   If the function execution takes longer than this value,
   the execution will be canceled and an error returned.
   Use `0` for no timeout.

**Returns**

A [`JsonResponse`](#jsonresponse-type).
If the execution is successful, the `result` field is set to `true`.

For example:

```json
{
   "result": true
}
```

### `extern char* waku_connect_peerid(char* peerId, int timeoutMs)`

Dial peer using its peer ID.

**Parameters**

1`char* peerID`: Peer ID to dial.
   The peer must be already known.
   It must have been added before with [`waku_add_peer`](#extern-char-waku_add_peerchar-address-char-protocolid)
   or previously dialed with [`waku_connect_peer`](#extern-char-waku_connect_peerchar-address-int-timeoutms).
2. `int timeoutMs`: Timeout value in milliseconds to execute the call.
   If the function execution takes longer than this value,
   the execution will be canceled and an error returned.
   Use `0` for no timeout.

**Returns**

A [`JsonResponse`](#jsonresponse-type).
If the execution is successful, the `result` field is set to `true`.

For example:

```json
{
   "result": true
}
```

### `extern char* waku_disconnect_peer(char* peerId)`

Disconnect a peer using its peerID

**Parameters**

1. `char* peerID`: Peer ID to disconnect.

**Returns**

A [`JsonResponse`](#jsonresponse-type).
If the execution is successful, the `result` field is set to `true`.

For example:

```json
{
   "result": true
}
```

### `extern char* waku_peer_count()`

Get number of connected peers.

**Returns**

A [`JsonResponse`](#jsonresponse-type).
If the execution is successful, the `result` field contains an `integer` which represents the number of connected peers.

For example:

```json
{
  "result": 0
}
```

### `extern char* waku_peers()`

Retrieve the list of peers known by the Waku node.

**Returns**

A [`JsonResponse`](#jsonresponse-type) containing a list of peers.
The list of peers has this format:

```json
{
  "result": [
    {
      "peerID": "16Uiu2HAmJb2e28qLXxT5kZxVUUoJt72EMzNGXB47RedcBafeDCBA",
      "protocols": [
        "/ipfs/id/1.0.0",
        "/vac/waku/relay/2.0.0",
        "/ipfs/ping/1.0.0"
      ],
      "addrs": [
        "/ip4/1.2.3.4/tcp/30303"
      ],
      "connected": true
    }
  ]
}
```

## Waku Relay

### `extern char* waku_content_topic(char* applicationName, unsigned int applicationVersion, char* contentTopicName, char* encoding)`

Create a content topic string according to [RFC 23](https://rfc.vac.dev/spec/23/).

**Parameters**

1. `char* applicationName`
2. `unsigned int applicationVersion`
3. `char* contentTopicName`
4. `char* encoding`: depending on the payload, use `proto`, `rlp` or `rfc26`

**Returns**

`char *` containing a content topic formatted according to [RFC 23](https://rfc.vac.dev/spec/23/).

```
/{application-name}/{version-of-the-application}/{content-topic-name}/{encoding}
```

### `extern char* waku_pubsub_topic(char* name, char* encoding)`

Create a pubsub topic string according to [RFC 23](https://rfc.vac.dev/spec/23/).

**Parameters**

1. `char* name`
2. `char* encoding`: depending on the payload, use `proto`, `rlp` or `rfc26`

**Returns**

`char *` containing a content topic formatted according to [RFC 23](https://rfc.vac.dev/spec/23/).

```
/waku/2/{topic-name}/{encoding}
```

### `extern char* waku_default_pubsub_topic()`

Returns the default pubsub topic used for exchanging waku messages defined in [RFC 10](https://rfc.vac.dev/spec/10/).

**Returns**

`char *` containing the default pubsub topic:

```
/waku/2/default-waku/proto
```

### `extern char* waku_relay_publish(char* messageJson, char* pubsubTopic, int timeoutMs)`

Publish a message using Waku Relay.

**Parameters**

1. `char* messageJson`: JSON string containing the [Waku Message](https://rfc.vac.dev/spec/14/) as [`JsonMessage`](#jsonmessage-type).
2. `char* pubsubTopic`: pubsub topic on which to publish the message.
   If `NULL`, it uses the default pubsub topic.
3. `int timeoutMs`: Timeout value in milliseconds to execute the call.
   If the function execution takes longer than this value,
   the execution will be canceled and an error returned.
   Use `0` for no timeout.

Note: `messageJson.version` is overwritten to `0`.

**Returns**

A [`JsonResponse`](#jsonresponse-type).
If the execution is successful, the `result` field contains the message ID.

### `extern char* waku_relay_publish_enc_asymmetric(char* messageJson, char* pubsubTopic, char* publicKey, char* optionalSigningKey, int timeoutMs)`

Optionally sign,
encrypt using asymmetric encryption
and publish a message using Waku Relay.

**Parameters**

1. `char* messageJson`: JSON string containing the [Waku Message](https://rfc.vac.dev/spec/14/) as [`JsonMessage`](#jsonmessage-type).
2. `char* pubsubTopic`: pubsub topic on which to publish the message.
   If `NULL`, it uses the default pubsub topic.
3. `char* publicKey`: hex encoded public key to be used for encryption.
4. `char* optionalSigningKey`: hex encoded private key to be used to sign the message.
5. `int timeoutMs`: Timeout value in milliseconds to execute the call.
   If the function execution takes longer than this value,
   the execution will be canceled and an error returned.
   Use `0` for no timeout.

Note: `messageJson.version` is overwritten to `1`.

**Returns**

A [`JsonResponse`](#jsonresponse-type).
If the execution is successful, the `result` field contains the message ID.

### `extern char* waku_relay_publish_enc_symmetric(char* messageJson, char* pubsubTopic, char* symmetricKey, char* optionalSigningKey, int timeoutMs)`

Optionally sign,
encrypt using symmetric encryption
and publish a message using Waku Relay.

**Parameters**

1. `char* messageJson`: JSON string containing the [Waku Message](https://rfc.vac.dev/spec/14/) as [`JsonMessage`](#jsonmessage-type).
2. `char* pubsubTopic`: pubsub topic on which to publish the message.
   If `NULL`, it uses the default pubsub topic.
3. `char* symmetricKey`: hex encoded secret key to be used for encryption.
4. `char* optionalSigningKey`: hex encoded private key to be used to sign the message.
5. `int timeoutMs`: Timeout value in milliseconds to execute the call.
   If the function execution takes longer than this value,
   the execution will be canceled and an error returned.
   Use `0` for no timeout.

Note: `messageJson.version` is overwritten to `1`.

**Returns**

A [`JsonResponse`](#jsonresponse-type).
If the execution is successful, the `result` field contains the message ID.

### `extern char* waku_relay_enough_peers(char* pubsubTopic)`

Determine if there are enough peers to publish a message on a given pubsub topic.

**Parameters**

1. `char* pubsubTopic`: Pubsub topic to verify.
   If `NULL`, it verifies the number of peers in the default pubsub topic.

**Returns**

A [`JsonResponse`](#jsonresponse-type).
If the execution is successful, the `result` field contains a `boolean` indicating whether there are enough peers.

For example:

```json
{
  "result": true
}
```

### `extern char* waku_relay_subscribe(char* topic)`

Subscribe to a Waku Relay pubsub topic to receive messages.

**Parameters**

1. `char* topic`: Pubsub topic to subscribe to. 
   If `NULL`, it subscribes to the default pubsub topic.

**Returns**

A [`JsonResponse`](#jsonresponse-type).
If the execution is successful, the `result` field is set to `true`.

For example:

```json
{
  "result": true
}
```

**Events**

When a message is received, a ``"message"` event` is emitted containing the message, pubsub topic, and node ID in which
the message was received.

The `event` type is [`JsonMessageEvent`](#jsonmessageevent-type).

For Example:

```json
{
  "type": "message",
  "event": {
    "subscriptionID": 1,
    "pubsubTopic": "/waku/2/default-waku/proto",
    "messageID": "0x6496491e40dbe0b6c3a2198c2426b16301688a2daebc4f57ad7706115eac3ad1",
    "wakuMessage": {
      "payload": "TODO",
      "contentTopic": "/my-app/1/notification/proto",
      "version": 1,
      "timestamp": 1647826358000000000
    }
  }
}
```

### `extern char* waku_relay_unsubscribe(char* topic)`

Closes the pubsub subscription to a pubsub topic. No more messages will be received
from this pubsub topic.

**Parameters**

1. `char* pusubTopic`: Pubsub topic to unsubscribe from.
  If `NULL`, unsubscribes from the default pubsub topic.

**Returns**

A [`JsonResponse`](#jsonresponse-type).
If the execution is successful, the `result` field is set to `true`.

For example:

```json
{
   "result": true
}
```


## Waku Filter

### `extern char* waku_filter_subscribe(char* filterJSON, char* peerID, int timeoutMs)`

Creates a subscription in a lightnode for messages that matches a content filter and optionally a [PubSub `topic`](https://github.com/libp2p/specs/blob/master/pubsub/README.md#the-topic-descriptor).

**Parameters**

1. `char* filterJSON`: JSON string containing the [`FilterSubscription`](#filtersubscription-type) to subscribe to.
2. `char* peerID`: Peer ID to subscribe to.
   The peer must be already known.
   It must have been added before with [`waku_add_peer`](#extern-char-waku_add_peerchar-address-char-protocolid)
   or previously dialed with [`waku_connect_peer`](#extern-char-waku_connect_peerchar-address-int-timeoutms).
   Use `NULL` to automatically select a node.
3. `int timeoutMs`: Timeout value in milliseconds to execute the call.
   If the function execution takes longer than this value,
   the execution will be canceled and an error returned.
   Use `0` for no timeout.

**Returns**

A [`JsonResponse`](#jsonresponse-type).
If the execution is successful, the `result` field is set to `true`.

For example:

```json
{
  "result": true
}
```

**Events**

When a message is received, a ``"message"` event` is emitted containing the message, pubsub topic, and node ID in which
the message was received.

The `event` type is [`JsonMessageEvent`](#jsonmessageevent-type).

For Example:

```json
{
  "type": "message",
  "event": {
    "subscriptionID": 1,
    "pubsubTopic": "/waku/2/default-waku/proto",
    "messageID": "0x6496491e40dbe0b6c3a2198c2426b16301688a2daebc4f57ad7706115eac3ad1",
    "wakuMessage": {
      "payload": "TODO",
      "contentTopic": "/my-app/1/notification/proto",
      "version": 1,
      "timestamp": 1647826358000000000
    }
  }
}
```

### `extern char* waku_filter_unsubscribe(char* filterJSON, int timeoutMs)`

Removes subscriptions in a light node matching a content filter and, optionally, a [PubSub `topic`](https://github.com/libp2p/specs/blob/master/pubsub/README.md#the-topic-descriptor).

**Parameters**

1. `char* filterJSON`: JSON string containing the [`FilterSubscription`](#filtersubscription-type).
2. `int timeoutMs`: Timeout value in milliseconds to execute the call.
   If the function execution takes longer than this value,
   the execution will be canceled and an error returned.
   Use `0` for no timeout.

**Returns**

A [`JsonResponse`](#jsonresponse-type).
If the execution is successful, the `result` field is set to `true`.

For example:

```json
{
  "result": true
}
```

## Waku Lightpush

### `extern char* waku_lightpush_publish(char* messageJSON, char* topic, char* peerID, int timeoutMs)`

Publish a message using Waku Lightpush.

**Parameters**

1. `char* messageJson`: JSON string containing the [Waku Message](https://rfc.vac.dev/spec/14/) as [`JsonMessage`](#jsonmessage-type).
2. `char* pubsubTopic`: pubsub topic on which to publish the message.
   If `NULL`, it uses the default pubsub topic.
3. `char* peerID`: Peer ID supporting the lightpush protocol.
   The peer must be already known.
   It must have been added before with [`waku_add_peer`](#extern-char-waku_add_peerchar-address-char-protocolid)
   or previously dialed with [`waku_connect_peer`](#extern-char-waku_connect_peerchar-address-int-timeoutms).
   Use `NULL` to automatically select a node.
3. `int timeoutMs`: Timeout value in milliseconds to execute the call.
   If the function execution takes longer than this value,
   the execution will be canceled and an error returned.
   Use `0` for no timeout.

Note: `messageJson.version` is overwritten to `0`.

**Returns**

A [`JsonResponse`](#jsonresponse-type).
If the execution is successful, the `result` field contains the message ID.

### `extern char* waku_lightpush_publish_enc_asymmetric(char* messageJson, char* pubsubTopic, char* peerID, char* publicKey, char* optionalSigningKey, int timeoutMs)`

Optionally sign,
encrypt using asymmetric encryption
and publish a message using Waku Lightpush.

**Parameters**

1. `char* messageJson`: JSON string containing the [Waku Message](https://rfc.vac.dev/spec/14/) as [`JsonMessage`](#jsonmessage-type).
2. `char* pubsubTopic`: pubsub topic on which to publish the message.
   If `NULL`, it uses the default pubsub topic.
3. `char* peerID`: Peer ID supporting the lightpush protocol.
   The peer must be already known.
   It must have been added before with [`waku_add_peer`](#extern-char-waku_add_peerchar-address-char-protocolid)
   or previously dialed with [`waku_connect_peer`](#extern-char-waku_connect_peerchar-address-int-timeoutms).
4. `char* publicKey`: hex encoded public key to be used for encryption.
5. `char* optionalSigningKey`: hex encoded private key to be used to sign the message.
6. `int timeoutMs`: Timeout value in milliseconds to execute the call.
   If the function execution takes longer than this value,
   the execution will be canceled and an error returned.
   Use `0` for no timeout.

Note: `messageJson.version` is overwritten to `1`.

**Returns**

A [`JsonResponse`](#jsonresponse-type).
If the execution is successful, the `result` field contains the message ID.

### `extern char* waku_lightpush_publish_enc_symmetric(char* messageJson, char* pubsubTopic, char* peerID, char* symmetricKey, char* optionalSigningKey, int timeoutMs)`

Optionally sign,
encrypt using symmetric encryption
and publish a message using Waku Lightpush.

**Parameters**

1. `char* messageJson`: JSON string containing the [Waku Message](https://rfc.vac.dev/spec/14/) as [`JsonMessage`](#jsonmessage-type).
2. `char* pubsubTopic`: pubsub topic on which to publish the message.
   If `NULL`, it uses the default pubsub topic.
3. `char* peerID`: Peer ID supporting the lightpush protocol.
   The peer must be already known.
   It must have been added before with [`waku_add_peer`](#extern-char-waku_add_peerchar-address-char-protocolid)
   or previously dialed with [`waku_connect_peer`](#extern-char-waku_connect_peerchar-address-int-timeoutms).
4. `char* symmetricKey`: hex encoded secret key to be used for encryption.
5. `char* optionalSigningKey`: hex encoded private key to be used to sign the message.
6. `int timeoutMs`: Timeout value in milliseconds to execute the call.
   If the function execution takes longer than this value,
   the execution will be canceled and an error returned.
   Use `0` for no timeout.

Note: `messageJson.version` is overwritten to `1`.

**Returns**

A [`JsonResponse`](#jsonresponse-type).
If the execution is successful, the `result` field contains the message ID.

## Waku Store

### `extern char* waku_store_query(char* queryJSON, char* peerID, int timeoutMs)`

Retrieves historical messages on specific content topics. This method may be called with [`PagingOptions`](#pagingoptions-type), 
to retrieve historical messages on a per-page basis. If the request included [`PagingOptions`](#pagingoptions-type), the node 
must return messages on a per-page basis and include [`PagingOptions`](#pagingoptions-type) in the response. These [`PagingOptions`](#pagingoptions-type) 
must contain a cursor pointing to the Index from which a new page can be requested.

**Parameters**

1. `char* queryJSON`: JSON string containing the [`StoreQuery`](#storequery-type).
2. `char* peerID`: Peer ID supporting the store protocol.
   The peer must be already known.
   It must have been added before with [`waku_add_peer`](#extern-char-waku_add_peerchar-address-char-protocolid)
   or previously dialed with [`waku_connect_peer`](#extern-char-waku_connect_peerchar-address-int-timeoutms).
3. `int timeoutMs`: Timeout value in milliseconds to execute the call.
   If the function execution takes longer than this value,
   the execution will be canceled and an error returned.
   Use `0` for no timeout.

**Returns**

A [`JsonResponse`](#jsonresponse-type).
If the execution is successful, the `result` field contains a [`StoreResponse`](#storeresponse-type)..


## Decrypting messages

### `extern char* waku_decode_symmetric(char* messageJson, char* symmetricKey)`
Decrypt a message using a symmetric key

**Parameters**

1. `char* messageJson`: JSON string containing the [Waku Message](https://rfc.vac.dev/spec/14/) as [`JsonMessage`](#jsonmessage-type).
2. `char* symmetricKey`: 32 byte symmetric key hex encoded.

Note: `messageJson.version` is expected to be `1`.

**Returns**

A [`JsonResponse`](#jsonresponse-type).
If the execution is successful, the `result` field contains the decoded payload as a [`DecodedPayload`](#decodedpayload-type).
An `error` message otherwise.

```json
{
  "result": {
    "pubkey": "0x......",
    "signature": "0x....",
    "data": "...",
    "padding": "..."
  }
}
```

### `extern char* waku_decode_asymmetric(char* messageJson, char* privateKey)`
Decrypt a message using a secp256k1 private key 

**Parameters**

1. `char* messageJson`: JSON string containing the [Waku Message](https://rfc.vac.dev/spec/14/) as [`JsonMessage`](#jsonmessage-type).
2. `char* privateKey`: secp256k1 private key hex encoded.

Note: `messageJson.version` is expected to be `1`.

**Returns**

A [`JsonResponse`](#jsonresponse-type).
If the execution is successful, the `result` field contains the decoded payload as a [`DecodedPayload`](#decodedpayload-type).
An `error` message otherwise.

```json
{
  "result": {
    "pubkey": "0x......",
    "signature": "0x....",
    "data": "...",
    "padding": "..."
  }
}
```

## Utils

### `extern char* waku_utils_base64_encode(char* data)`

Encode a byte array to base64.
Useful for creating the payload of a Waku Message in the format understood by [`waku_relay_publish`](#extern-char-waku_relay_publishchar-messagejson-char-pubsubtopic-int-timeoutms)

**Parameters**

1. `char* data`: Byte array to encode

**Returns**

A `char *` containing the base64 encoded byte array.

### `extern char* waku_utils_base64_decode(char* data)`

Decode a base64 string (useful for reading the payload from Waku Messages).

**Parameters**

1. `char* data`: base64 encoded byte array to decode.

**Returns**

A [`JsonResponse`](#jsonresponse-type).
If the execution is successful, the `result` field contains the decoded payload.

# Copyright

Copyright and related rights waived via
[CC0](https://creativecommons.org/publicdomain/zero/1.0/).
