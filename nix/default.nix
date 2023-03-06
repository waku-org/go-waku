#{ pkgs ? import <nixpkgs> {
{ pkgs ? import ../../nixpkgs {
  config.android_sdk.accept_license = true;
} }:

{
  pkg = pkgs.callPackage ./mobile_pkg.nix { };
  mod = pkgs.callPackage ./mobile_mod.nix { };
  bkp = pkgs.callPackage ./mobile_bkp.nix { };
}
