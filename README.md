# go-waku
A Go implementation of the [Waku v2 protocol](https://specs.vac.dev/specs/waku/v2/waku-v2).


## Waku Protocol Support

- ✔: Supported
- 🚧: Implementation in progress
- ⛔: Support is not planned

| Spec | Implementation Status |
| ---- | -------------- |
|[7/WAKU-DATA](https://rfc.vac.dev/spec/7)|✔|
|[10/WAKU2](https://rfc.vac.dev/spec/10)|🚧|
|[11/WAKU2-RELAY](https://rfc.vac.dev/spec/11)|✔|
|[12/WAKU2-FILTER](https://rfc.vac.dev/spec/12)||
|[13/WAKU2-STORE](https://rfc.vac.dev/spec/13)|🚧|
|[14/WAKU2-MESSAGE](https://rfc.vac.dev/spec/14)|✔|
|[15/WAKU2-BRIDGE](https://rfc.vac.dev/spec/15)|⛔|
|[16/WAKU2-RPC](https://rfc.vac.dev/spec/16)||
|[17/WAKU2-RLNRELAY](https://rfc.vac.dev/spec/17)||
|[18/WAKU2-SWAP](https://rfc.vac.dev/spec/18)||


## Install
```
git clone https://github.com/status-im/go-waku
cd go-waku
make
```

## Wakunode
See the available command line options with
```
./build/waku --help
```


## Examples
Examples of usage of go-waku as a library can be found in the examples folder. There is a fully featured chat example.


## Contribution
Thank you for considering to help out with the source code! We welcome contributions from anyone on the internet, and are grateful for even the smallest of fixes!

If you'd like to contribute to go-waku, please fork, fix, commit and send a pull request. If you wish to submit more complex changes though, please check up with the core devs first to ensure those changes are in line with the general philosophy of the project and/or get some early feedback which can make both your efforts much lighter as well as our review and merge procedures quick and simple.

To build and test this repository, you need:
  - [Go](https://golang.org/)
  - [protoc](https://grpc.io/docs/protoc-installation/) 
  - [Protocol Buffers for Go with Gadgets](https://github.com/gogo/protobuf)
]

## License
Licensed and distributed under either of

* MIT license: [LICENSE-MIT](LICENSE-MIT) or http://opensource.org/licenses/MIT

or

* Apache License, Version 2.0, ([LICENSE-APACHEv2](LICENSE-APACHEv2) or http://www.apache.org/licenses/LICENSE-2.0)

at your option. These files may not be copied, modified, or distributed except according to those terms.
