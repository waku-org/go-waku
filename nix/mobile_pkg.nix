{ lib, pkgs, callPackage, buildGoModule, buildGoPackage }:

let
  androidPkgs = pkgs.androidenv.composeAndroidPackages {
    toolsVersion = "26.1.1";
    platformToolsVersion = "33.0.1";
    buildToolsVersions = [ "31.0.0" ];
    platformVersions = [ "30" ];
    cmakeVersions = [ "3.18.1" ];
    ndkVersion = "22.1.7171670";
    includeNDK = true;
  };
  androidSdk = androidPkgs.androidsdk;
  gomobile = pkgs.gomobile.override { inherit androidPkgs; };
in buildGoPackage rec {
  pname = "go-waku";
  version = "devel";
  goPackagePath = "github.com/status-im/go-waku";

  src = ./..;

  extraSrcPaths = [ gomobile ];
  nativeBuildInputs = [ gomobile pkgs.openjdk8 ];

  ANDROID_HOME = "${androidSdk}/libexec/android-sdk";

  buildPhase = ''
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
