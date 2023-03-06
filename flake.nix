{
  description = "Nix flake for Go implementaion of Waku v2 node.";

  inputs.nixpkgs.url = github:NixOS/nixpkgs/nixos-22.11;

  outputs = { self, nixpkgs }:
    let
      supportedSystems = [ "x86_64-linux" "i686-linux" "aarch64-linux" ];
      forAllSystems = f: nixpkgs.lib.genAttrs supportedSystems (system: f system);

      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });

      buildPackage = system: subPackages:
        let
          pkgs = nixpkgsFor.${system};
          commit = builtins.substring 0 7 (self.rev or "dirty");
          version = builtins.readFile ./VERSION;
        in pkgs.buildGo119Module {
          name = "go-waku";
          src = self;
          inherit subPackages;
          ldflags = [
            "-X github.com/waku-org/go-waku/waku/v2/node.GitCommit=${commit}"
            "-X github.com/waku-org/go-waku/waku/v2/node.Version=${version}"
          ];
          doCheck = false;
          # FIXME: This needs to be manually changed when updating modules.
          vendorSha256 = "sha256-TvQfLQEYDujfXInQ+i/LoSGtedozZvX8WgzpqiryYHY=";
          # Fix for 'nix run' trying to execute 'go-waku'.
          meta = { mainProgram = "waku"; };
        };
    in rec {
      packages = forAllSystems (system: {
        node    = buildPackage system ["cmd/waku"];
        library = buildPackage system ["library"];
      });

      defaultPackage = forAllSystems (system:
        buildPackage system ["cmd/waku"]
      );

      devShells.default = forAllSystems (system:
        packages.${system}.node
      );
  };
}
