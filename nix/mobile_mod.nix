{ pkgs, buildGo119Module }:

let
  androidPkgs = pkgs.androidenv.composeAndroidPackages {
    platformVersions = [ "23" ];
    ndkVersion = "22.1.7171670";
    includeNDK = true;
  };
  androidSdk = androidPkgs.androidsdk;
  #gomobile = pkgs.gomobile.override { inherit androidPkgs; };
in buildGo119Module {
  pname = "go-waku";
  version = "devel";
  goPkgPath = "github.com/status-im/go-waku";
  vendorSha256 = "sha256-tNBIxJv9Vty83UQ31XWzXicRVcEAIs30eyA3RBQz9nw=";
  doCheck = false;
  #proxyVendor = true;

  src = ./..;

  nativeBuildInputs = with pkgs; [ gomobile openjdk8 ];

  ANDROID_HOME = "${androidSdk}/libexec/android-sdk";
  #GOFLAGS = "-mod=mod";
  #GO111MODULE = "on";

  #overrideModAttrs = (_: {
  #  postBuild = ''
  #    echo 'WTF ------------------------------------------------------'
  #    go install golang.org/x/mobile/cmd/gomobile
  #    echo 'WTF ------------------------------------------------------'
  #  '';
  #});

  # Correct GOPATH necessary to avoid error:
  # `no exported names in the package "_/build/go-waku/mobile"`
  postConfigure = ''
    echo "$GOPATH"
    cd "$NIX_BUILD_TOP"
    mkdir -p "go/src/$(dirname "$goPkgPath")"
    mv "$sourceRoot" "go/src/$goPkgPath"
    cd go/src/$goPkgPath
  '';

  preBuild = ''
    echo '-----------------------------------------------'
    find -L vendor | grep go-ethereum
    echo '-----------------------------------------------'
    find -L vendor | grep prometheus
    echo '-----------------------------------------------'
    ls -l vendor/contrib.go.opencensus.io/exporter/prometheus
    echo '-----------------------------------------------'
    ls -l vendor/contrib.go.opencensus.io/exporter/prometheus/@v
    echo '-----------------------------------------------'
    ls -l vendor/contrib.go.opencensus.io/exporter/prometheus/@v/list
    echo '-----------------------------------------------'
  '';
  buildPhase = ''
    runHook preBuild

    gomobile bind -x -v \
      -target=android/arm64 \
      -androidapi=23 \
      -ldflags="-s -w" \
      -o go-waku.aar \
      ./mobile

    runHook postBuild
  '';

  installPhase = ''
    mkdir -p $out
    mv go-waku.aar $out/
  '';
}
