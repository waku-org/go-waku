#{ pkgs ? import <nixpkgs> {
{ pkgs ? import ../../nixpkgs {
  config.android_sdk.accept_license = true;
} }:

#pkgs.callPackage ./mobile_pkg.nix { }
pkgs.callPackage ./mobile_mod.nix { }
