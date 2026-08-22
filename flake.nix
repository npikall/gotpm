{
  description = "GoTPM development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
  }:
    flake-utils.lib.eachDefaultSystem (
      system: let
        pkgs = nixpkgs.legacyPackages.${system};
      in {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Go toolchain
            go_1_27

            # Development tools
            gopls # Go language server
            golangci-lint # Linter
            gofumpt # Formatter (stricter than gofmt)
            go-task # Taskrunner

            # Additional tools
            git # Version control
            gh # GitHub CLI
            svu # Semantic version utility
          ];

          shellHook = ''
            if [ -n "$ZSH_VERSION" ]; then
              export PROMPT="%F{green}(nix)%f %~ %# "
            else
              export PS1="\[\033[1;32m\](nix)\[\033[0m\] \w \$ "
            fi

            # Set Go environment variables
            export CGO_ENABLED=0
          '';
        };
      }
    );
}
