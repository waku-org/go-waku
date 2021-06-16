# Using the `peer_events` application

## Background

The `peer_events` application is a basic example app that demonstrates how peer event handling works in go-waku

## Preparation
```
make
```

## Basic application usage

To start the `peer_events` application run the following from the project directory

```
./build/peer_events
```
The app will run the following nodes sequentially:
- relayNode1 and relayNode2
- relayNode2 is stopped, and relayNode3 is started
- relayNode3 is stopped, and storeNode is started

