{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs";
    flake-utils.url = "github:numtide/flake-utils";
  };
  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        packages = {
          default = pkgs.buildGoModule {
            pname = "go-torrent-client";
            version = "0-unstable-2025-06-13";
            src = ./.;
            vendorHash = null;
            env.CGO_ENABLED = "0";
            ldflags = [ "-s" ];
          };
        };
      }
    );
}
