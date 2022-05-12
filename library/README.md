---
slug: #
title: #/GOWAKU-BINDINGS
name: Go-Waku v2 C Bindings
status: draft
tags: go-waku
editor: Richard Ramos <richard@status.im>
contributors:

---

This specification describes the API for consuming go-waku when built as a dynamic or static library


# libgowaku.h


## General


### JSONResponse
All the API functions return a `JSONResponse` unless specified otherwise. `JSONResponse` is a `char *` whose format depends on whether the function was executed sucessfully or not:
```js
// On failure:
{ "error": "the error message" }

// On success:
{ "result": ...  } // result format depends on the function response
```

## Events
Asynchronous events require a callback to be registered. An example of an asynchronous event that might be emitted is receiving a message. When an event is emitted, this callback will be triggered receiving a json string with the following format:
```js
{
	"type": "message", // type of signal being emitted. Currently only "message" is available
	"event": ... // format depends on the type of signal. In the case of "message", a waku message can be expected here
}
```

### `extern void waku_set_event_callback(void* cb)`
Register callback to act as signal handler and receive application signals, which are used to react to asyncronous events in waku. 
**Parameters**
1. `void* cb`: callback that will be executed when an async event is emitted. The function signature for the callback should be `void myCallback(char* signalJSON)`


## Node management

### `extern char* waku_new(char* configJSON)`
Initialize a go-waku node.

**Parameters**
1. `char* configJSON`: JSON string containing the options used to initialize a go-waku node. It can be `NULL` to use defaults. All the keys from the configuration are optional. If a key is `undefined`, or `null`, a default value will be set 
    ```js
    // example config:
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
    - `host` - `String` (optional): Listening IP address. Default `0.0.0.0`
    - `port` - `Number` (optional): Libp2p TCP listening port. Default `60000`. Use `0` for random
    - `advertiseAddr` - `String` (optional): External address to advertise to other nodes. 
    - `nodeKey` - `String` (optional): secp256k1 private key in Hex format (`0x123...abc`). Default random
    - `keepAliveInterval` - `Number` (optional): Interval in seconds for pinging peers to keep the connection alive. Default `20`
    - `relay` - `Boolean` (optional): Enable relay protocol. Default `true`
    - `minPeersToPublish` - `Number` (optional). The minimum number of peers required on a topic to allow broadcasting a message. Default `0`

**Returns**
`JSONResponse` with a NULL `result`. An `error` message otherwise

---

### `extern char* waku_start()`
Initialize a go-waku node mounting all the protocols that were enabled during the waku node initialization.

**Returns**
`JSONResponse` containing a null `result` if the function executes successfully. An `error` message otherwise

---

### `extern char* waku_stop()`
Stops a go-waku node

**Returns**
`JSONResponse` containing a null `result` if the function executes successfully. An `error` message otherwise

---

### `extern char* waku_peerid()`
Obtain the peer ID of the go-waku node.

**Returns**
`JSONResponse` containing the peer ID (base58 encoded) if the function executes successfully. An `error` message otherwise

---

### `extern char* waku_listen_addresses()`
Obtain the multiaddresses the wakunode is listening to

**Returns**
`JSONResponse` containing an array of multiaddresses if the function executes successfully. An `error` message otherwise


## Connecting to peers

### `extern char* waku_add_peer(char* address, char* protocolID)`
Add node multiaddress and protocol to the wakunode peerstore

**Parameters**
1. `char* address`: multiaddress of the peer being added
2. `char* protocolID`: protocol supported by the peer

**Returns**
`JSONResponse` containing the peer ID (base58 encoded) of the peer that was added if the function executes successfully. An `error` message otherwise

---

### `extern char* waku_connect(char* address, int ms)`
Connect to peer at multiaddress.

**Parameters**
1. `char* address`: multiaddress of the peer being dialed
2. `int ms`: max duration in milliseconds this function might take to execute. If the function execution takes longer than this value, the execution will be canceled and an error returned. Use `0` for unlimited duration

**Returns**
`JSONResponse` with a null `result` if the function executes successfully. An `error` message otherwise

---

### `extern char* waku_connect_peerid(char* id, int ms)`
Connect to peer using peerID.

**Parameters**
1. `char* peerID`: peerID to dial. The peer must be already known. It must have been added before with `waku_add_peer` or previously dialed with `waku_dial_peer`
2. `int ms`: max duration in milliseconds this function might take to execute. If the function execution takes longer than this value, the execution will be canceled and an error returned. Use `0` for unlimited duration

**Returns**
`JSONResponse` with a null `result` if the function executes successfully. An `error` message otherwise

---

### `extern char* waku_disconnect(char* peerID)`
Disconnect a peer using its peerID. 

**Parameters**
1. `char* peerID`: peerID to disconnect.

**Returns**
`JSONResponse` with a null `result` if the function executes successfully. An `error` message otherwise

---

### `extern char* waku_peer_cnt()`
Obtain number of connected peers

**Returns**
`JSONResponse` containing an `int` with the number of connected peers. An `error` message otherwise

---

### `extern char* waku_peers()`
Retrieve the list of peers known by the go-waku node

**Returns**
`JSONResponse` containing a list of peers. An `error` message otherwise. The list of peers has this format:
```js
{
    "result":[
        ...
        {
            "peerID":"16Uiu2HAmJb2e28qLXxT5kZxVUUoJt72EMzNGXB47RedcBafeDCBA",
            "protocols":[
                "/ipfs/id/1.0.0",
                "/vac/waku/relay/2.0.0",
                "/ipfs/ping/1.0.0",
                ...
            ],
            "addrs":[
                "/ip4/1.2.3.4/tcp/30303",
                ...
            ],
            "connected":true
        }
    ]
}
```


## Waku Relay


### `extern char* waku_content_topic(char* applicationName, unsigned int applicationVersion, char* contentTopicName, char* encoding)`
Create a content topic string according to [RFC 23](https://rfc.vac.dev/spec/23/)

**Parameters**
1. `char* applicationName`
2. `unsigned int applicationVersion`
3. `char* contentTopicName`
4. `char* encoding`: depending on the payload, use `proto`, `rlp` or `rfc26`

**Returns**
`char *` containing a content topic formatted according to [RFC 23](https://rfc.vac.dev/spec/23/) 
```
/{application-name}/{version-of-the-application}/{content-topic-name}/{encoding}
```

--

### `extern char* waku_pubsub_topic(char* name, char* encoding)`
Create a pubsub topic string according to [RFC 23](https://rfc.vac.dev/spec/23/)

**Parameters**
1. `char* name`
2. `char* encoding`: depending on the payload, use `proto`, `rlp` or `rfc26`

**Returns**
`char *` containing a content topic formatted according to [RFC 23](https://rfc.vac.dev/spec/23/) 
```
/waku/2/{topic-name}/{encoding}
```

---

### `extern char* waku_default_pubsub_topic()`
Returns the default pubsub topic used for exchanging waku messages defined in [RFC 10](https://rfc.vac.dev/spec/10/)

**Returns**
`char *` containing the default pubsub topic:
```
/waku/2/default-waku/proto
```

---

### `extern char* waku_relay_publish(char* messageJSON, char* topic, int ms)`
Publish a message using waku relay. 

**Parameters**
1. `char* messageJSON`: json string containing the [Waku Message](https://rfc.vac.dev/spec/14/)
    ```js
    {
        "payload":"", // base64 encoded payload. waku_utils_base64_encode can be used for this
        "contentTopic: "...",
        "version": 1,
        "timestamp": 1647963508000000000 // Unix timestamp in nanoseconds
    }
    ```
3. `char* topic`: pubsub topic. Set to `NULL` to  use the default pubsub topic
4. `int ms`: max duration in milliseconds this function might take to execute. If the function execution takes longer than this value, the execution will be canceled and an error returned. Use `0` for unlimited duration

**Returns**
`JSONResponse` containing the message ID. An `error` message otherwise

---

### `extern char* waku_relay_publish_enc_asymmetric(char* messageJSON, char* topic, char* publicKey, char* optionalSigningKey, int ms)`
Publish a message encrypted with a secp256k1 public key using waku relay

**Parameters**
1. `char* messageJSON`: json string containing the [Waku Message](https://rfc.vac.dev/spec/14/)
    ```js
    {
        "payload":"", // base64 encoded payload. waku_utils_base64_encode can be used for this
        "contentTopic: "...",
        "version": 1,
        "timestamp": 1647963508000000000 // Unix timestamp in nanoseconds
    }
    ```
2. `char* topic`: pubsub topic. Set to `NULL` to  use the default pubsub topic
3. `char* publicKey`: hex string prefixed with "0x" containing a valid secp256k1 public key.
4. `char* optionalSigningKey`:  optional hex string prefixed with "0x" containing a valid secp256k1 private key for signing the message. Use NULL otherwise
5. `int ms`: max duration in milliseconds this function might take to execute. If the function execution takes longer than this value, the execution will be canceled and an error returned. Use `0` for unlimited duration

**Returns**
`JSONResponse` containing the message ID. An `error` message otherwise

---

### `extern char* waku_relay_publish_enc_symmetric(char* messageJSON, char* topic, char* symmetricKey, char* optionalSigningKey, int ms)`
Publish a message encrypted with a 32 bytes symmetric key using waku relay

**Parameters**
1. `char* messageJSON`: json string containing the [Waku Message](https://rfc.vac.dev/spec/14/)
    ```js
    {
        "payload":"", // base64 encoded payload. waku_utils_base64_encode can be used for this
        "contentTopic: "...",
        "version": 1,
        "timestamp": 1647963508000000000 // Unix timestamp in nanoseconds
    }
    ```
2. `char* topic`: pubsub topic. Set to `NULL` to  use the default pubsub topic
3. `char* symmetricKey`: hex string prefixed with "0x" containing a 32 bytes symmetric key
4. `char* optionalSigningKey`:  optional hex string prefixed with "0x" containing a valid secp256k1 private key for signing the message. Use NULL otherwise
5. `int ms`: max duration in milliseconds this function might take to execute. If the function execution takes longer than this value, the execution will be canceled and an error returned. Use `0` for unlimited duration

**Returns**
`JSONResponse` containing the message ID. An `error` message otherwise

---

### `extern char* waku_enough_peers(char* topic)`
Determine if there are enough peers to publish a message on a topic.

**Parameters**
1. `char* topic`: pubsub topic to verify. Use `NULL` to verify the number of peers in the default pubsub topic

**Returns**
`JSONResponse` with a boolean indicating if there are enough peers or not. An `error` message otherwise

---

### `extern char* waku_relay_subscribe(char* topic)`
Subscribe to a WakuRelay topic to receive messages. 

**Parameters**
1. `char* topic`: pubsub topic to subscribe to. Use `NULL` for subscribing to the default pubsub topic


**Returns**
`JSONResponse` with a null result. An `error` message otherwise

**Events**
When a message is received, a ``"message"` event` is emitted containing the message, and pubsub topic in which the message was received. Here's an example event that could be received:
```js
{
  "type":"message",
  "event":{
    "pubsubTopic":"/waku/2/default-waku/proto",
    "messageID":"0x6496491e40dbe0b6c3a2198c2426b16301688a2daebc4f57ad7706115eac3ad1",
    "wakuMessage":{
      "payload":"...", // base64 encoded message. Use waku_decode_data to decode
      "contentTopic":"ABC",
      "version":1,
      "timestamp":1647826358000000000 // in nanoseconds
    }
  }
}
```

---

### `extern char* waku_relay_unsubscribe(char* topic)`
Closes the subscription to a pubsub topic.

**Parameters**
1. `char* topic`: pubsub topic to unsubscribe from. Use `NULL` for unsubscribe from the default pubsub topic

**Returns**
`JSONResponse` with null `response` if successful. An `error` message otherwise


## Waku LightPush


### `extern char* waku_lightpush_publish(char* messageJSON, char* topic, char* peerID, int ms)`
Publish a message using waku lightpush. 

**Parameters**
1. `char* messageJSON`: json string containing the [Waku Message](https://rfc.vac.dev/spec/14/)
    ```js
    {
        "payload":"", // base64 encoded payload. waku_utils_base64_encode can be used for this
        "contentTopic: "...",
        "version": 1,
        "timestamp": 1647963508000000000 // Unix timestamp in nanoseconds
    }
    ```
2. `char* topic`: pubsub topic. Set to `NULL` to  use the default pubsub topic
3. `char* peerID`: should contain the ID of a peer supporting the lightpush protocol. Use NULL to automatically select a node
4. `int ms`: max duration in milliseconds this function might take to execute. If the function execution takes longer than this value, the execution will be canceled and an error returned. Use `0` for unlimited duration

**Returns**
`JSONResponse` containing the message ID. An `error` message otherwise

---

### `extern char* waku_lightpush_publish_enc_asymmetric(char* messageJSON, char* topic, char* publicKey, char* optionalSigningKey, char* peerID, int ms)`
Publish a message encrypted with a secp256k1 public key using waku lightpush

**Parameters**
1. `char* messageJSON`: json string containing the [Waku Message](https://rfc.vac.dev/spec/14/)
    ```js
    {
        "payload":"", // base64 encoded payload. waku_utils_base64_encode can be used for this
        "contentTopic: "...",
        "version": 1,
        "timestamp": 1647963508000000000 // Unix timestamp in nanoseconds
    }
    ```
2. `char* topic`: pubsub topic. Set to `NULL` to  use the default pubsub topic
3. `char* publicKey`: hex string prefixed with "0x" containing a valid secp256k1 public key.
4. `char* optionalSigningKey`:  optional hex string prefixed with "0x" containing a valid secp256k1 private key for signing the message. Use NULL otherwise
5. `char* peerID`: should contain the ID of a peer supporting the lightpush protocol. Use NULL to automatically select a node
6. `int ms`: max duration in milliseconds this function might take to execute. If the function execution takes longer than this value, the execution will be canceled and an error returned. Use `0` for unlimited duration

**Returns**
`JSONResponse` containing the message ID. An `error` message otherwise

---

### `extern char* waku_lightpush_publish_enc_symmetric(char* messageJSON, char* topic, char* symmetricKey, char* optionalSigningKey, char* peerID, int ms)`
Publish a message encrypted with a 32 bytes symmetric key using waku relay

**Parameters**
1. `char* messageJSON`: json string containing the [Waku Message](https://rfc.vac.dev/spec/14/)
    ```js
    {
        "payload":"", // base64 encoded payload. waku_utils_base64_encode can be used for this
        "contentTopic: "...",
        "version": 1,
        "timestamp": 1647963508000000000 // Unix timestamp in nanoseconds
    }
    ```
2. `char* topic`: pubsub topic. Set to `NULL` to  use the default pubsub topic
3. `char* symmetricKey`: hex string prefixed with "0x" containing a 32 bytes symmetric key
4. `char* optionalSigningKey`:  optional hex string prefixed with "0x" containing a valid secp256k1 private key for signing the message. Use NULL otherwise
5. `char* peerID`: should contain the ID of a peer supporting the lightpush protocol. Use NULL to automatically select a node
6. `int ms`: max duration in milliseconds this function might take to execute. If the function execution takes longer than this value, the execution will be canceled and an error returned. Use `0` for unlimited duration

**Returns**
`JSONResponse` containing the message ID. An `error` message otherwise


## Waku Store


### `extern char* waku_store_query(char* queryJSON, char* peerID, int ms)`
Query historic messages using waku store protocol.

**Parameters**
1. `char* queryJSON`: json string containing the query. If the message length is greater than 0, this function should be executed again, setting  the `cursor` attribute with the cursor returned in the response
```js
{
  "pubsubTopic": "...", // optional string
  "startTime": 1234, // optional, unix epoch time in nanoseconds
  "endTime": 1234, // optional, unix epoch time in nanoseconds
  "contentFilters": [ // optional
    {
      "contentTopic": "..."
    }, ...
  ],
  "pagingOptions": { // optional pagination information
    "pageSize": 40, // number
    "cursor": { // optional
      "digest": ...,
      "receiverTime": ...,
      "senderTime": ...,
      "pubsubTopic" ...,
    },
    "forward": true, // sort order
  }
}
```
2. `char* peerID`: should contain the ID of a peer supporting the store protocol. Use NULL to automatically select a node
3. `int ms`: max duration in milliseconds this function might take to execute. If the function execution takes longer than this value, the execution will be canceled and an error returned. Use `0` for unlimited duration

**Returns**
`JSONResponse` containing the store response. An `error` message otherwise
```js
{
  "result": {
    "messages": [ ... ], // array of waku messages
    "pagingOptions": { // optional pagination information
      "pageSize": 40, // number
      "cursor": { // optional
        "digest": ...,
        "receiverTime": ...,
        "senderTime": ...,
        "pubsubTopic" ...,
      },
      "forward": true, // sort order
    }
  }
}
```

## Decrypting messages

### `extern char* waku_decode_symmetric(char* messageJSON, char* symmetricKey)`
Decrypt a message using a symmetric key

**Parameters**
1. `char* messageJSON`: json string containing the [Waku Message](https://rfc.vac.dev/spec/14/)
    ```js
    {
        "payload":"...", // encrypted payload encoded in base64.
        "contentTopic: "...",
        "version": 1,
        "timestamp": 1647963508000000000 // Unix timestamp in nanoseconds
    }
    ```
2. `char* symmetricKey`: 32 byte symmetric key

**Returns**
`JSONResponse` containing a `DecodedPayload`. An `error` message otherwise
```js
{
  "result": {
    "pubkey": "0x......", // pubkey that signed the message (optional)
    "signature": "0x....", // message signature (optional)
    "data": "...", // decrypted message payload encoded in base64
    "padding": "...", // base64 encoded padding
  }
}

```

### `extern char* waku_decode_asymmetric(char* messageJSON, char* privateKey)`
Decrypt a message using a secp256k1 private key 

**Parameters**
1. `char* messageJSON`: json string containing the [Waku Message](https://rfc.vac.dev/spec/14/)
    ```js
    {
        "payload":"...", // encrypted payload encoded in base64.
        "contentTopic: "...",
        "version": 1,
        "timestamp": 1647963508000000000 // Unix timestamp in nanoseconds
    }
    ```
2. `char* privateKey`: secp256k1 private key 

**Returns**
`JSONResponse` containing a `DecodedPayload`. An `error` message otherwise
```js
{
  "result": {
    "pubkey": "0x......", // pubkey that signed the message (optional)
    "signature": "0x....", // message signature (optional)
    "data": "...", // decrypted message payload encoded in base64
    "padding": "...", // base64 encoded padding
  }
}
```

## Waku Message Utils


### `extern char* waku_utils_base64_encode(char* data)`
Encode a byte array to base64 useful for creating the payload of a waku message in the format understood by `waku_relay_publish`

**Parameters**
1. `char* data`: byte array to encode
 
**Returns**
A `char *` containing the base64 encoded byte array

---

### `extern char* waku_utils_base64_decode(char* data)`
Decode a base64 string (useful for reading the payload from waku messages)

**Parameters**
1. `char* data`: base64 encoded byte array to decode
 
**Returns**
`JSONResponse` with the decoded payload. An `error` message otherwise. The decoded payload has this format:

---
### `extern void waku_utils_free(char* data)`
Frees a char* since all strings returned by gowaku are allocated in the C heap using malloc.

**Parameters**
1. `char* data`: variable to free


# Copyright

Copyright and related rights waived via
[CC0](https://creativecommons.org/publicdomain/zero/1.0/).
