# go-waku
A Go implementation of the [Waku v2 protocol](https://specs.vac.dev/specs/waku/v2/waku-v2).

<p align="left">
  <a href="https://goreportcard.com/report/github.com/status-im/go-waku"><img src="https://goreportcard.com/badge/github.com/status-im/go-waku" /></a>
  <a href="https://godoc.org/github.com/status-im/go-waku"><img src="http://img.shields.io/badge/godoc-reference-5272B4.svg?style=flat-square" /></a>
  <a href=""><img src="https://img.shields.io/badge/golang-%3E%3D1.15.0-orange.svg?style=flat-square" /></a>
  <br>
</p>

## Install

#### Building from source
```
git clone https://github.com/status-im/go-waku
cd go-waku
make

# See the available command line options with
./build/waku --help
```

#### Docker
```
docker build -t go-waku:latest .

docker run go-waku:latest --help
```

## Library
```
go get github.com/status-im/go-waku
```

## Examples
Examples of usage of go-waku as a library can be found in the examples folder. There is a fully featured chat example.


## Waku Protocol Support

- ✔: Supported
- 🚧: Implementation in progress
- ⛔: Support is not planned

| Spec | Implementation Status |
| ---- | -------------- |
|[7/WAKU2](https://rfc.vac.dev/spec/7)|✔|
|[10/WAKU2](https://rfc.vac.dev/spec/10)|🚧|
|[11/WAKU2-RELAY](https://rfc.vac.dev/spec/11)|✔|
|[12/WAKU2-FILTER](https://rfc.vac.dev/spec/12)|✔|
|[13/WAKU2-STORE](https://rfc.vac.dev/spec/13)|✔|
|[14/WAKU2-MESSAGE](https://rfc.vac.dev/spec/14)|✔|
|[15/WAKU2-BRIDGE](https://rfc.vac.dev/spec/15)|⛔|
|[16/WAKU2-RPC](https://rfc.vac.dev/spec/16)||
|[17/WAKU2-RLNRELAY](https://rfc.vac.dev/spec/17)||
|[18/WAKU2-SWAP](https://rfc.vac.dev/spec/18)||
|[21/WAKU2-FTSTORE](https://rfc.vac.dev/spec/21)|✔|
|[22/TOY-CHAT](https://rfc.vac.dev/spec/22)|✔|
|[23/TOPICS](https://rfc.vac.dev/spec/22)|(implemented in status-go)|
|[25/LIBP2P-DNS-DISCOVERY](https://rfc.vac.dev/spec/25)|🚧|
|[26/WAKU2-PAYLOAD](https://rfc.vac.dev/spec/26)|✔|
|[27/WAKU2-PEERS](https://rfc.vac.dev/spec/27)|✔|
|[29/WAKU2-CONFIG](https://rfc.vac.dev/spec/29)|🚧|

## Contribution
Thank you for considering to help out with the source code! We welcome contributions from anyone on the internet, and are grateful for even the smallest of fixes!

If you'd like to contribute to go-waku, please fork, fix, commit and send a pull request. If you wish to submit more complex changes though, please check up with the core devs first to ensure those changes are in line with the general philosophy of the project and/or get some early feedback which can make both your efforts much lighter as well as our review and merge procedures quick and simple.

To build and test this repository, you need:
  - [Go](https://golang.org/) (version 1.15 or later)
  - [protoc](https://grpc.io/docs/protoc-installation/) 
  - [Protocol Buffers for Go with Gadgets](https://github.com/gogo/protobuf)

To enable the git hooks:

```bash
git config core.hooksPath hooks
```

## License
Licensed and distributed under either of

* MIT license: [LICENSE-MIT](LICENSE-MIT) or http://opensource.org/licenses/MIT

or

* Apache License, Version 2.0, ([LICENSE-APACHEv2](LICENSE-APACHEv2) or http://www.apache.org/licenses/LICENSE-2.0)

at your option. These files may not be copied, modified, or distributed except according to those terms.
