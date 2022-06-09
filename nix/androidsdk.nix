{ stdenv, androidenv }:

# The "android-sdk-license" license is accepted
# by setting android_sdk.accept_license = true.
androidenv.composeAndroidPackages {
  toolsVersion = "26.1.1";
  platformToolsVersion = "31.0.3";
  buildToolsVersions = [ "31.0.0" ];
  platformVersions = [ "30" ];
  cmakeVersions = [ "3.18.1" ];
  ndkVersion = "22.1.7171670";
  includeNDK = true;
  includeExtras = [
    "extras;android;m2repository"
    "extras;google;m2repository"
  ];
}
# This derivation simply symlinks some stuff to get
# shorter paths as libexec/android-sdk is quite the mouthful.
# With this you can just do `androidPkgs.sdk` and `androidPkgs.ndk`.
#stdenv.mkDerivation {
#  name = "${compose.androidsdk.name}-mod";
#  phases = [ "symlinkPhase" ];
#  outputs = [ "out" "sdk" "ndk" ];
#  symlinkPhase = ''
#    ln -s ${compose.androidsdk} $out
#    ln -s ${compose.androidsdk}/libexec/android-sdk $sdk
#    ln -s ${compose.androidsdk}/libexec/android-sdk/ndk-bundle $ndk
#  '';
#}
