# Using the `basic_light_client` application

## Background

The `basic_light_client` application is a basic example app that demonstrates how to send/receive messages using Waku Filter and Lightpush.

There are 2 ways of running the example.:
1. To work with the public Waku network in which case it uses the autosharding feature.This is the default way to run this.
2. To work with a custom Waku network which using static sharding. In this case a clusterID has to be specified.

## Preparation
```
make
```

## Basic application usage

To start the `basic_light_client` application run the following from the project directory

```
./build/basic_light_client --maddr=<filterNode>
```

The app will send a "Hello world!" through the lightpush protocol and display it on the terminal as soon as it receives the message.

In order to run it with you own static sharded network, then run it as below

```
./build/basic_light_client  --maddr=<filterNode> --cluster-id=<value of cluster-id> --shard=<shard number> 
```
e.g: ./build/basic_light_client --maddr="/ip4/0.0.0.0/tcp/30304/p2p/16Uiu2HAmBu5zRFzBGAzzMAuGWhaxN2EwcbW7CzibELQELzisf192" --cluster-id=2 --shard=1   // If you want to run with clusterID 2 and shard as 1

Cluster-id is a unique identifier for your own network and shard number is a segment/shard identifier of your network.

Note that clusterID's 1 & 16 are reserved for the public Waku Network and Status repectively.