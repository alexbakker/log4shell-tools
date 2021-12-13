{
  description = "Nix flake for log4shell-tools";
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-21.11";

  outputs = { self, nixpkgs }: let
      pkgs = import nixpkgs { system = "x86_64-linux"; };
    in {
      defaultPackage.x86_64-linux =
        with pkgs; buildGoModule {
          pname = "log4shell-tools";
          version = "0.0.0";
          src = ./.;

          vendorSha256 = "sha256-qDtzwInCe3fJY7EENTxN5g23deFz6FcLGUZbgwyBtyQ=";

          doCheck = false;

          subPackages = [ "./cmd/log4shell-tools-server" ];
        };
      nixosConfigurations.devContainer = nixpkgs.lib.nixosSystem {
        system = "x86_64-linux";
        modules = [
          ./container.nix
        ];
      };
      devShell.x86_64-linux = with pkgs; mkShell {
        buildInputs = [
          go
          maven
          openjdk8
        ];
      };
    };
}
