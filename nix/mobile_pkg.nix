{ lib, pkgs, callPackage, buildGo119Package }:

let
  androidPkgs = pkgs.androidenv.composeAndroidPackages {
    platformVersions = [ "23" ];
    ndkVersion = "22.1.7171670";
    includeNDK = true;
  };
  androidSdk = androidPkgs.androidsdk;
  gomobile = pkgs.gomobile.override { inherit androidPkgs; };
in buildGo119Package rec {
  pname = "go-waku";
  version = "devel";
  goPackagePath = "github.com/status-im/go-waku";

  src = ./..;

  extraSrcPaths = [ gomobile ];
  nativeBuildInputs = [ gomobile pkgs.openjdk8 ];

  ANDROID_HOME = "${androidSdk}/libexec/android-sdk";

  buildPhase = ''
    echo GOPATH: $GOPATH
    echo PWD: $PWD
    ls -l go/src/${goPackagePath}
    set -x
    gomobile bind -x \
      -target=android/arm64 \
      -androidapi=23 \
      -ldflags="-s -w" \
      -o go-waku.aar \
      ${goPackagePath}/mobile
  '';

  installPhase = ''
    mkdir -p $out
    mv go-waku.aar $out/
  '';

  #buildPhase = ''
  #  echo $ANDROID_HOME
  #  gomobile bind -x \
  #    -target=ios \
  #    -iosversion=8.0 \
  #    -ldflags="-s -w" \
  #    -o ./build/lib/go-waku.xcframework \
  #    ./mobile
  #'';
}
