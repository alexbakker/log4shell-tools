{
  description = "Nix flake for log4shell-tools";
  inputs.nixpkgs.url = "nixpkgs/nixos-unstable";
  inputs.flake-utils.url = "github:numtide/flake-utils";

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
      in {
        defaultPackage =
          with pkgs; buildGoModule {
            name = "log4shell-tools";
            src = ./.;

            vendorHash = "sha256-TjtI04iYvpist+BH7Abrpru5cs0kmC0VJlX1HBT/tjw=";

            subPackages = [ "./cmd/log4shell-tools-server" ];
          };
        nixosConfigurations.devContainer = nixpkgs.lib.nixosSystem {
          inherit system;
          modules = [
            ./container.nix
          ];
        };
        devShell = with pkgs; mkShell {
          buildInputs = [
            go
            maven
            openjdk8
          ];
        };
      }
    );
}
