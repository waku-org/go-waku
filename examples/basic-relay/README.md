# Using the `basic_relay` application

## Background

The `basic_relay` application is a basic example app that demonstrates how to subscribe to and publish messages using Waku relay.

There are 2 ways of running the example.:
1. To work with the public Waku network in which case it uses the autosharding feature.This is the default way to run this.
2. To work with a custom Waku network which using static sharding. In this case a clusterID has to be specified.

## Preparation
```
make
```

## Basic application usage

To start the `basic_relay` application run the following from the project directory

```
./build/basic_relay
```

The app will send a "Hello world!" through the wakurelay protocol every 2 seconds and display it on the terminal as soon as it receives the message.

In order to run it with you own static sharded network, then run it as below

```
./build/basic_relay --cluster-id=<value of cluster-id> --shard=<shard number>
```
e.g: ./build/basic_relay --cluster-id=2 --shard=1   // If you want to run with clusterID 2 and shard as 1

Cluster-id is a unique identifier for your own network and shard number is a segment/shard identifier of your network.

Note that clusterID's 1 & 16 are reserved for the public Waku Network and Status repectively.