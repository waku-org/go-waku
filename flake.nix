{
  description = "Nix flake for Go implementaion of Waku v2 node.";

  inputs.nixpkgs.url = github:NixOS/nixpkgs/nixos-23.11;

  outputs = { self, nixpkgs }:
    let
      supportedSystems = [
        "x86_64-linux" "i686-linux" "aarch64-linux"
        "x86_64-darwin" "aarch64-darwin"
      ];
      forAllSystems = f: nixpkgs.lib.genAttrs supportedSystems (system: f system);

      pkgsFor = forAllSystems (system: import nixpkgs { inherit system; });

      buildPackage = system: subPackages:
        let
          pkgs = pkgsFor.${system};
          commit = builtins.substring 0 7 (self.rev or "dirty");
          version = builtins.readFile ./VERSION;
        in pkgs.buildGo121Module {
          name = "go-waku";
          src = self;
          inherit subPackages;
          tags = [ ];
          ldflags = [
            "-X github.com/waku-org/go-waku/waku/v2/node.GitCommit=${commit}"
            "-X github.com/waku-org/go-waku/waku/v2/node.Version=${version}"
          ];
          doCheck = false;
          # FIXME: This needs to be manually changed when updating modules.
          vendorHash = "sha256-+IcLqxWNCxYnPqFz3CA9dpPvlRVsxVxvQUJ9eNehKAU=";
          # Fix for 'nix run' trying to execute 'go-waku'.
          meta = { mainProgram = "waku"; };
        };
    in rec {
      packages = forAllSystems (system: let
        pkgs = pkgsFor.${system};
        os = pkgs.stdenv.hostPlatform.uname.system;
        sttLibExtMap = { Windows = "lib"; Darwin = "a";     Linux = "a";  };
        dynLibExtMap = { Windows = "dll"; Darwin = "dylib"; Linux = "so"; };
        buildPackage = pkgs.callPackage ./default.nix;
      in rec {
        default = node;
        node = buildPackage {
          inherit self;
          subPkgs = ["cmd/waku"];
        };
        static-library = buildPackage {
          inherit self;
          subPkgs = ["library/c"];
          ldflags = ["-buildmode=c-archive"];
          output = "libgowaku.${sttLibExtMap.${os}}";
        };
        # FIXME: Compilation fails with:
        #   relocation R_X86_64_TPOFF32 against runtime.tlsg can not be
        #   used when making a shared object; recompile with -fPIC
        dynamic-library = buildPackage {
          inherit self;
          subPkgs = ["library/c"];
          ldflags = ["-buildmode=c-shared"];
          output = "libgowaku.${dynLibExtMap.${os}}";
        };
      });

      devShells = forAllSystems (system: let
        pkgs = pkgsFor.${system};
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
