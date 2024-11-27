# Mobile

Package mobile implements [gomobile](https://github.com/golang/mobile) bindings for go-waku. 

## Usage

For proper use this package, please refer to Makefile in the root of `go-waku` directory.

To manually build library, run following commands:

### iOS

```
gomobile init
gomobile bind -v -target=ios -ldflags="-s -w" github.com/waku-org/go-waku/mobile
```
This will produce `gowaku.framework` file in the current directory, which can be used in iOS project.

### Android

```
export ANDROID_NDK_HOME=/path/to/android/ndk
export ANDROID_HOME=/path/to/android/sdk/
gomobile init
gomobile bind -v -target=android -ldflags="-s -w" github.com/waku-org/go-waku/mobile
```
This will generate `gowaku.aar` file in the current directory.

## Notes

See [https://github.com/golang/go/wiki/Mobile](https://github.com/golang/go/wiki/Mobile) for more information on `gomobile` usage.
