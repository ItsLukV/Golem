{
  description = "Minecraft server implementation in Go";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "golem";
          version = "0.1.0";
          src = ./.;
          vendorHash = "sha256-Q2QH7qLz13nxSuyfMtpdnc3OIhwU599b4kOSbp821Wo=";
        };

        devShells.default = pkgs.mkShell {
          packages = [ pkgs.go pkgs.gopls ];
        };
      });
}
