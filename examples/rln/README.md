# Using the `rln` example application

## Background

The `rln` application is a basic example app that demonstrates how to subscribe to and publish messages using Waku Relay with RLN

## Preparation

Edit `main.go` and set proper values to these constants and variables:
```go
const ethClientAddress = "wss://sepolia.infura.io/ws/v3/API_KEY_GOES_HERE"
const ethPrivateKey = "PRIVATE_KEY_GOES_HERE"
const contractAddress = "0xF471d71E9b1455bBF4b85d475afb9BB0954A29c4"
const credentialsPath = ""
const credentialsPassword = ""

var contentTopic = protocol.NewContentTopic("rln", 1, "test", "proto").String()
var pubsubTopic = protocol.DefaultPubsubTopic{}
```
The private key used here should contain enough Sepolia ETH to register on the contract (0.001 ETH). An ethereum client address is required as well. After updating these values, execute `make`

## Basic application usage

To start the `rln` application run the following from the project directory

```
./build/rln
```

The app will send a "Hello world!" through the waku relay protocol every 2 seconds and display it on the terminal as soon as it receives the message. Only a single message per epoch is allowed, so 4 out of 5 messages will be considered spam.
