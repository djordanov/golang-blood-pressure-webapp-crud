{
  description = "A simple blood pressure CRUD web app in Golang";

  inputs.nixpkgs.url = "nixpkgs/nixos-26.05";

  outputs = { self, nixpkgs, ... }:
    let
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
    in
    {
      packages.${system} = {
        bpo = pkgs.buildGoModule {
          pname = "bpo";
          version = "0.2.0";
          src = ./.;
          vendorHash = "sha256-zNiOubWRZOTJFyhcmM+lKdI954hfEgrpikM8pIyabTk=";
        };
        container = pkgs.dockerTools.buildImage {
          name = "bpo-container";
          tag = "latest";
          contents = [ self.packages.${system}.bpo];
          config = {
            Cmd = [ "${self.packages.${system}.bpo}/bin/bpo"];
            ExposedPorts = { "8080/tcp" = {}; };
          };
        };

        default = self.packages.${system}.bpo;
      };

      devShells.${system}.default = pkgs.mkShell {
        packages = with pkgs; [ go sqlc gopls gotools go-tools delve air postgresql ];
        shellHook = "echo 'Entered Flake'";
      };
    };
}
