{ pkgs ? import ../../nixpkgs {
  config.android_sdk.accept_license = true;
} }:

pkgs.callPackage ./mobile.nix { }
