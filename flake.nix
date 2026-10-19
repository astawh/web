{
  description = "AStA WH website and development environment";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      forAllSystems = nixpkgs.lib.genAttrs [
        "aarch64-darwin" "x86_64-darwin" "aarch64-linux" "x86_64-linux"
      ];
      settings = system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          version = builtins.head (builtins.match ".*\ngo ([0-9.]+)\n.*" (builtins.readFile ./go.mod));
          go = pkgs.${"go_" + builtins.replaceStrings [ "." ] [ "_" ] (pkgs.lib.versions.majorMinor version)};
        in { inherit pkgs go; };
    in {
      packages = forAllSystems (system:
        let inherit (settings system) pkgs go;
        in {
          default = (pkgs.buildGoModule.override { inherit go; }) {
            pname = "asta-web";
            version = "0.1.0";
            src = self;
            vendorHash = null;
            env.CGO_ENABLED = "0";
            ldflags = [ "-s" "-w" "-X main.Version=0.1.0" "-X main.Revision=${self.rev or "dirty"}" ];
          };
        });
      devShells = forAllSystems (system:
        let inherit (settings system) pkgs go;
        in { default = pkgs.mkShell { packages = [ go pkgs.actionlint ]; }; });
      checks = forAllSystems (system: { build = self.packages.${system}.default; });
    };
}
