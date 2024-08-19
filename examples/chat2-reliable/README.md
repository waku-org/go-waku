# chat2-reliable: A Reliable P2P Chat Application

## Background

`chat2-reliable` is an enhanced version of a basic command-line chat application that uses the [Waku v2 suite of protocols](https://specs.vac.dev/specs/waku/v2/waku-v2). This version implements an end-to-end reliability protocol to ensure message delivery and causal ordering in a distributed environment.

## Features

- P2P chat capabilities using Waku v2 protocols
- Implementation of e2e reliability protocol
- Support for group chats and direct communication
- Scalable to large groups (up to 10K participants)
- Transport-agnostic design

## E2E Reliability Protocol

The e2e reliability protocol in `chat2-reliable` is an implementation of the proposal at [Vac Forum](https://forum.vac.dev/t/end-to-end-reliability-for-scalable-distributed-logs/293) and includes the following key features:

1. **Lamport Clocks**: Each participant maintains a Lamport clock for logical timestamping of messages.

2. **Causal History**: Messages include a short causal history (preceding message IDs) to establish causal relationships.

3. **Bloom Filters**: A rolling bloom filter is used to track received message IDs and detect duplicates.

4. **Lazy Pull Mechanism**: Missing messages are requested from peers when causal dependencies are unmet.

5. **Eager Push Mechanism**: Unacknowledged messages are resent to ensure delivery.

## Usage

### Building the Application

```
make
```

### Starting the Application

Basic usage:
```
./build/chat2-reliable
```

With custom DNS server:
```
./build/chat2-reliable --dns-discovery-name-server 8.8.8.8
```

### In-chat Commands

- `/help`: Display available commands
- `/connect`: Interactively connect to a new peer
- `/peers`: Display the list of connected peers

Example:
```
/connect /ip4/127.0.0.1/tcp/58426/p2p/16Uiu5rGt2QDLmPKas9zpsBgtr5kRzk473s9wkKSWoYwfcY4Hco33
```

## Message Format

Messages in `chat2-reliable` use the following protobuf format:

```protobuf
message Message {
  string sender_id = 1;
  string message_id = 2;
  int32 lamport_timestamp = 3;
  repeated string causal_history = 4;
  string channel_id = 5;
  bytes bloom_filter = 6;
  string content = 7;
}
```

## Implementation Details

1. **Lamport Clocks**: Implemented in the `Chat` struct with methods to increment, update, and retrieve the timestamp.

2. **Causal History**: Stored in the `CausalHistory` field of each message, containing IDs of recent preceding messages.

3. **Bloom Filters**: Implemented as a `RollingBloomFilter` to efficiently track received messages and detect duplicates.

4. **Message Processing**:
   - Incoming messages are checked against the bloom filter for duplicates.
   - Causal dependencies are verified before processing.
   - Messages with unmet dependencies are stored in an incoming buffer.

5. **Message Recovery**:
   - Missing messages are requested from peers.
   - A retry mechanism with exponential backoff is implemented for failed retrievals.

6. **Conflict Resolution**:
   - Messages are ordered based on Lamport timestamps and message IDs for consistency.

7. **Periodic Tasks**:
   - Buffer sweeps to process buffered messages and resend unacknowledged ones.
   - Sync messages to maintain consistency across peers.

## Testing

The implementation includes various tests to ensure the reliability features work as expected:

- Lamport timestamp correctness
- Causal ordering of messages
- Duplicate detection using bloom filters
- Message recovery after network partitions
- Concurrent message sending
- Large group scaling
- Eager push mechanism effectiveness
- Bloom filter window functionality
- Conflict resolution
- New node synchronization

```
go test -v
```