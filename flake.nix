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

        lux = pkgs.buildGoApplication {
          pname = "lux";
          inherit version;
          src = ./.;
          modules = ./gomod2nix.toml;
          subPackages = [ "cmd/lux" ];

          meta = with pkgs.lib; {
            description = "LSP Multiplexer that routes requests to language servers based on file type";
            homepage = "https://github.com/friedenberg/lux";
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

        apps.install-mcp = {
          type = "app";
          program = toString (
            pkgs.writeShellScript "install-lux-mcp" ''
              set -euo pipefail

              CLAUDE_CONFIG_DIR="''${HOME}/.claude"
              MCP_CONFIG_FILE="''${CLAUDE_CONFIG_DIR}/mcp.json"

              log() {
                ${pkgs.gum}/bin/gum style --foreground 212 "$1"
              }

              log_success() {
                ${pkgs.gum}/bin/gum style --foreground 82 "✓ $1"
              }

              log_error() {
                ${pkgs.gum}/bin/gum style --foreground 196 "✗ $1"
              }

              # Create config directory if needed
              if [[ ! -d "$CLAUDE_CONFIG_DIR" ]]; then
                log "Creating $CLAUDE_CONFIG_DIR..."
                mkdir -p "$CLAUDE_CONFIG_DIR"
              fi

              # Build the flake reference
              FLAKE_REF="${self}"

              # New MCP server entry
              NEW_SERVER=$(${pkgs.jq}/bin/jq -n \
                --arg cmd "nix" \
                --arg flake "$FLAKE_REF" \
                '{command: $cmd, args: ["run", $flake]}')

              if [[ -f "$MCP_CONFIG_FILE" ]]; then
                log "Found existing MCP config at $MCP_CONFIG_FILE"

                # Check if lux server already exists
                if ${pkgs.jq}/bin/jq -e '.mcpServers.lux' "$MCP_CONFIG_FILE" > /dev/null 2>&1; then
                  if ${pkgs.gum}/bin/gum confirm "lux MCP server already configured. Overwrite?"; then
                    UPDATED=$(${pkgs.jq}/bin/jq --argjson server "$NEW_SERVER" '.mcpServers.lux = $server' "$MCP_CONFIG_FILE")
                    echo "$UPDATED" > "$MCP_CONFIG_FILE"
                    log_success "Updated lux MCP server configuration"
                  else
                    log "Skipping installation"
                    exit 0
                  fi
                else
                  UPDATED=$(${pkgs.jq}/bin/jq --argjson server "$NEW_SERVER" '.mcpServers.lux = $server' "$MCP_CONFIG_FILE")
                  echo "$UPDATED" > "$MCP_CONFIG_FILE"
                  log_success "Added lux MCP server to existing configuration"
                fi
              else
                log "Creating new MCP config at $MCP_CONFIG_FILE"
                ${pkgs.jq}/bin/jq -n --argjson server "$NEW_SERVER" '{mcpServers: {lux: $server}}' > "$MCP_CONFIG_FILE"
                log_success "Created MCP configuration"
              fi

              log ""
              log "Installation complete! The lux MCP server will be available in Claude Code."
              log "Configuration written to: $MCP_CONFIG_FILE"
              log ""
              log "To verify, run: cat $MCP_CONFIG_FILE"
            ''
          );
        };
      }
    );
}
