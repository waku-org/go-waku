{ lib, pkgs, callPackage, buildGoModule }:

let
  androidPkgs = pkgs.androidenv.composeAndroidPackages {
    platformVersions = [ "23" ];
    ndkVersion = "22.1.7171670";
    includeNDK = true;
  };
  androidSdk = androidPkgs.androidsdk;
  #gomobile = pkgs.gomobile.override { inherit androidPkgs; };
in buildGoModule {
  pname = "go-waku";
  version = "devel";
  vendorSha256 = "sha256-+U8mlEK7HvCtssI8VDqxGaOvi+IU0+cr4isZLl9sB4o=";
  deleteVendor = true;
  doCheck = false;

  src = ./..;

  nativeBuildInputs = [ pkgs.gomobile pkgs.openjdk8 ];

  # We can't symlink gomobile src in vendor created by buildGoModule.
  proxyVendor = true;

  ANDROID_HOME = "${androidSdk}/libexec/android-sdk";

  # Correct GOPATH necessary to avoid error:
  # `no exported names in the package "_/build/go-waku/mobile"`
  preBuild = ''
    export GO111MODULE=off
    mkdir -p /build/go/src/github.com/status-im
    mv /build/go-waku /build/go/src/github.com/status-im/
    cd /build/go/src/github.com/status-im/go-waku
  '';

  buildPhase = ''
    runHook preBuild
    gomobile bind -x \
      -target=android/arm64 \
      -androidapi=23 \
      -ldflags="-s -w" \
      -o go-waku.aar \
      ./mobile
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
