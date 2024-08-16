{
  pkgs ? import <nixpkgs> { },
  self ? ./.,
  subPkgs ? "cmd/waku",
  ldflags ? [],
  output ? null,
  commit ? builtins.substring 0 7 (self.rev or "dirty"),
  version ? builtins.readFile ./VERSION,
}:

pkgs.buildGo121Module {
  name = "go-waku";
  src = self;

  subPackages = subPkgs;
  tags = ["gowaku_no_rln"];
  ldflags = [
    "-X github.com/waku-org/go-waku/waku/v2/node.GitCommit=${commit}"
    "-X github.com/waku-org/go-waku/waku/v2/node.Version=${version}"
  ] ++ ldflags;
  doCheck = false;

  # Otherwise library would be just called bin/c.
  postInstall = if builtins.isString output then ''
    mv $out/bin/* $out/bin/${output}
  '' else "";

  # FIXME: This needs to be manually changed when updating modules.
  vendorHash = "sha256-cOh9LNmcaBnBeMFM1HS2pdH5TTraHfo8PXL37t/A3gQ=";

  # Fix for 'nix run' trying to execute 'go-waku'.
  meta = { mainProgram = "waku"; };
}
