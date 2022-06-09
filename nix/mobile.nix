{ lib, pkgs, callPackage, buildGoModule }:

let
  androidPkgs = pkgs.androidenv.composeAndroidPackages {
    includeNDK = true;
    ndkVersion = "22.1.7171670";
  };
  androidSdk = androidPkgs.androidsdk;
  gomobile = pkgs.gomobile.override { inherit androidPkgs; };
in buildGoModule {
  pname = "go-waku";
  version = "devel";
  vendorSha256 = "sha256-+W5PnVmD4oPh3a8Ik9Xn3inCI8shqEsdlkG/d6PQntk=";
  GIT_COMMIT = "TODO";
  doCheck = false;

  src = ./..;

  extraSrcPaths = [ gomobile ];
  nativeBuildInputs = [ gomobile pkgs.openjdk8 ];

  # We can't symlink gomobile src in vendor created by buildGoModule.
  proxyVendor = true;

  ANDROID_HOME = "${androidSdk}/libexec/android-sdk";
  ANDROID_NDK_ROOT = "${androidSdk}/libexec/android-sdk/ndk-bundle";
  ANDROID_NDK_HOME = "${androidSdk}/libexec/android-sdk/ndk-bundle";
  #GOMOBILE = pkgs.gomobile;
  #GOHOSTARCH = "amd64";
  #GO111MODULE = "off";

  buildPhase = ''
    gomobile bind -x \
      -target=android/arm64 \
      -androidapi=23 \
      -ldflags="-s -w" \
      -o ./build/lib/Gowaku.xcframework \
      ./mobile
  '';

  #buildPhase = ''
  #  echo $ANDROID_HOME
  #  gomobile bind -x \
  #    -target=ios \
  #    -iosversion=8.0 \
  #    -ldflags="-s -w" \
  #    -o ./build/lib/Gowaku.xcframework \
  #    ./mobile
  #'';
}
