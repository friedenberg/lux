{
  description = "Lux: LSP Multiplexer";

  inputs = {
    nixpkgs-master.url = "github:NixOS/nixpkgs/b28c4999ed71543e71552ccfd0d7e68c581ba7e9";
    nixpkgs.url = "github:NixOS/nixpkgs/23d72dabcb3b12469f57b37170fcbc1789bd7457";
    utils.url = "https://flakehub.com/f/numtide/flake-utils/0.1.102";
    go.url = "github:friedenberg/eng?dir=devenvs/go";
    shell.url = "github:friedenberg/eng?dir=devenvs/shell";
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
