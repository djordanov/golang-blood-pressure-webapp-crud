{
  description = "A simple CRUD web app in Golang";

  inputs.nixpkgs.url = "nixpkgs/nixos-26.05";

  outputs = { nixpkgs, ... }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
    in
    {
      packages.${system} = {
        default = pkgs.buildGoModule {
          pname = "bpo";
          version = "0.1.0";
          src = ./.;
          vendorHash = "sha256-zNiOubWRZOTJFyhcmM+lKdI954hfEgrpikM8pIyabTk=";
        };
      };

      devShells.${system}.default = pkgs.mkShell {
        packages = with pkgs; [ go sqlc gopls gotools go-tools delve air ];
        shellHook = "echo 'Entered Flake'";
      };
    };
}
