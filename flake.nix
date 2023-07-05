{
  description = "Nix flake for Go implementaion of Waku v2 node.";

  inputs.nixpkgs.url = github:NixOS/nixpkgs/master;

  outputs = { self, nixpkgs }:
    let
      supportedSystems = [
        "x86_64-linux" "i686-linux" "aarch64-linux"
        "x86_64-darwin" "aarch64-darwin"
      ];
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
          vendorSha256 = "sha256-TU/jog0MZNC4g13gaGm88gsKTRvmlcKkMeXZbaVf3fc=";
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

      devShells = forAllSystems (system: let
        pkgs = nixpkgsFor.${system};
        inherit (pkgs) lib stdenv mkShell;
      in {
        default = mkShell {
          GOFLAGS = "-trimpath"; # Drop -mod=vendor
          inputsFrom = [ packages.${system}.node ];
          buildInputs = with pkgs; [ golangci-lint ];
          nativeBuildInputs = lib.optional stdenv.isDarwin [
            (pkgs.xcodeenv.composeXcodeWrapper { version = "14.2"; allowHigher = true; })
          ];
        };

        fpm = mkShell {
          buildInputs = with pkgs; [ fpm ];
        };
      });
  };
}
