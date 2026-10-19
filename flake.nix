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
          # renovate: datasource=npm depName=govuk-frontend
          govukVersion = "6.6.0";
          # renovate: datasource=github-tags depName=alphagov/govuk_publishing_components
          componentsVersion = "70.2.1";
          govuk = pkgs.fetchurl {
            url = "https://registry.npmjs.org/govuk-frontend/-/govuk-frontend-${govukVersion}.tgz";
            hash = "sha256-UsTrVU5U2KbkuoAenaXsOluTD9LxF6OWd/C2KW6VR1E=";
          };
          govukLicense = pkgs.fetchurl {
            url = "https://raw.githubusercontent.com/alphagov/govuk-frontend/v${govukVersion}/LICENSE.txt";
            hash = "sha256-xlLzWx8uW4jrSfHoqTXux5tHYagWZqzlduGvcj9Kkjg=";
          };
          components = pkgs.fetchurl {
            url = "https://codeload.github.com/alphagov/govuk_publishing_components/tar.gz/refs/tags/v${componentsVersion}";
            hash = "sha256-g0Nl3cNQqnJ/7PH7ABX130c86MUCr7aC4KABAE3glwo=";
          };
          assets = pkgs.stdenvNoCC.mkDerivation {
            pname = "asta-browser-assets";
            version = "1";
            dontUnpack = true;
            nativeBuildInputs = [ pkgs.dart-sass ];
            buildPhase = ''
              mkdir govuk components
              tar -xzf ${govuk} -C govuk --strip-components=1
              tar -xzf ${components} -C components --strip-components=1
              sass --no-source-map --style=compressed --quiet-deps \
                --load-path=govuk/dist \
                ${./view/styles/govuk.scss} govuk.css
              sass --no-source-map --style=compressed ${./view/styles/site.scss} site.css
            '';
            installPhase = ''
              mkdir -p "$out"
              install -m 0644 govuk.css site.css "$out/"
              install -m 0644 govuk/dist/govuk/govuk-frontend.min.js "$out/govuk.js"
              install -m 0644 components/app/assets/javascripts/govuk_publishing_components/components/layout-super-navigation-header.js "$out/super-navigation.js"
              install -m 0644 ${./view/site.js} "$out/site.js"
              install -m 0644 ${govukLicense} "$out/GOVUK-LICENSE.txt"
              install -m 0644 components/LICENCE.md "$out/GOVUK-COMPONENTS-LICENCE.md"
            '';
          };
        in {
          inherit assets;
          default = (pkgs.buildGoModule.override { inherit go; }) {
            pname = "asta-web";
            version = "0.1.0";
            src = self;
            vendorHash = "sha256-Pgi13UQT5JyTqTzImjH/M9wuDX2per4AfRJd7+jdp38=";
            env.CGO_ENABLED = "0";
            preBuild = ''
              cp -R ${assets}/. view/assets/
            '';
            ldflags = [ "-s" "-w" "-X main.Version=0.1.0" "-X main.Revision=${self.rev or "dirty"}" ];
          };
        });
      devShells = forAllSystems (system:
        let inherit (settings system) pkgs go;
        in { default = pkgs.mkShell { packages = [ go pkgs.actionlint ]; }; });
      apps = forAllSystems (system:
        let
          inherit (settings system) pkgs;
          vendor = pkgs.writeShellApplication {
            name = "vendor-assets";
            runtimeInputs = [ pkgs.coreutils ];
            text = ''
              test -f view/styles/govuk.scss
              cp -R --no-preserve=mode,ownership ${self.packages.${system}.assets}/. view/assets/
            '';
          };
        in { vendor = { type = "app"; program = "${vendor}/bin/vendor-assets"; }; });
      checks = forAllSystems (system:
        let inherit (settings system) pkgs;
        in {
          build = self.packages.${system}.default;
          assets = pkgs.runCommand "check-browser-assets" { } ''
            for file in ${self.packages.${system}.assets}/*; do
              diff "$file" "${self}/view/assets/$(basename "$file")"
            done
            touch "$out"
          '';
        });
    };
}
