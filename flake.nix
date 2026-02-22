{
  description = "Lux: LSP Multiplexer";

  inputs = {
    nixpkgs-master.url = "github:NixOS/nixpkgs/5b7e21f22978c4b740b3907f3251b470f466a9a2";
    nixpkgs.url = "github:NixOS/nixpkgs/6d41bc27aaf7b6a3ba6b169db3bd5d6159cfaa47";
    utils.url = "https://flakehub.com/f/numtide/flake-utils/0.1.102";
    go.url = "github:amarbel-llc/eng?dir=devenvs/go";
    shell.url = "github:amarbel-llc/eng?dir=devenvs/shell";
  };

  outputs =
    {
      self,
      nixpkgs,
      utils,
      go,
      shell, nixpkgs-master,
    }:
    utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [
            go.overlays.default
          ];
        };

        version = "0.1.0";

        manDocSrc = ./doc;

        lux = pkgs.buildGoApplication {
          pname = "lux";
          inherit version;
          src = ./.;
          modules = ./gomod2nix.toml;
          subPackages = [ "cmd/lux" ];

          nativeBuildInputs = [ pkgs.scdoc ];

          ldflags = [ "-X main.version=${version}" ];

          postInstall = ''
            # Generate plugin manifest, manpages (section 1), and completions
            $out/bin/lux _generate $out

            # Section 5: compile scdoc source (not handled by command.App)
            mkdir -p $out/share/man/man5
            scdoc < ${manDocSrc}/lux-config.5.scd > $out/share/man/man5/lux-config.5
          '';


          meta = with pkgs.lib; {
            description = "LSP Multiplexer that routes requests to language servers based on file type";
            homepage = "https://github.com/amarbel-llc/lux";
            license = licenses.mit;
          };
        };
      in
      {
        packages = {
          default = lux;
          inherit lux;
        };

        devShells.default = pkgs.mkShell {
          packages = with pkgs; [
            just
          ];

          inputsFrom = [
            go.devShells.${system}.default
            shell.devShells.${system}.default
          ];

          shellHook = ''
            echo "Lux: LSP Multiplexer - dev environment"
          '';
        };

        apps.default = {
          type = "app";
          program = "${lux}/bin/lux";
        };
      }
    );
}
