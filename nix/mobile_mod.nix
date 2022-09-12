{ pkgs, buildGoModule }:

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
  goPkgPath = "github.com/status-im/go-waku";
  vendorSha256 = "sha256-+W5PnVmD4oPh3a8Ik9Xn3inCI8shqEsdlkG/d6PQntk=";
  doCheck = false;
  proxyVendor = true;

  src = ./..;

  nativeBuildInputs = with pkgs; [ gomobile openjdk8 ];

  ANDROID_HOME = "${androidSdk}/libexec/android-sdk";
  #GOFLAGS = "-mod=mod";

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
    cd "$NIX_BUILD_TOP"
    mkdir -p "go/src/$(dirname "$goPkgPath")"
    mv "$sourceRoot" "go/src/$goPkgPath"
    cd go/src/$goPkgPath/mobile
  '';

  buildPhase = ''
    GO111MODULE=off gomobile bind -x \
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
}
