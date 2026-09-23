{
  description = "GoTPM - a minimal package manager for Typst";

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
        buildGoModule = pkgs.buildGoModule.override {go = pkgs.go_1_27;};
        # Flakes don't expose git tags, so the commit is the best version we have.
        rev = self.shortRev or self.dirtyShortRev or "dirty";
      in {
        packages = rec {
          gotpm = buildGoModule {
            pname = "gotpm";
            version = rev;
            src = self;

            # The repository ships a vendor/ directory.
            vendorHash = null;

            env.CGO_ENABLED = 0;
            nativeCheckInputs = [pkgs.git];
            ldflags = [
              "-s"
              "-w"
              "-X github.com/npikall/gotpm/cmd.gitTag=${rev}"
              "-X github.com/npikall/gotpm/cmd.gitCommit=${rev}"
              "-X github.com/npikall/gotpm/cmd.buildOS=${pkgs.go.GOOS}"
              "-X github.com/npikall/gotpm/cmd.buildARCH=${pkgs.go.GOARCH}"
              "-X github.com/npikall/gotpm/cmd.installer=nix"
            ];

            meta = {
              description = "A minimal package manager for Typst";
              homepage = "https://github.com/npikall/gotpm";
              license = pkgs.lib.licenses.mit;
              mainProgram = "gotpm";
            };
          };
          default = gotpm;
        };

        apps.default = flake-utils.lib.mkApp {drv = self.packages.${system}.gotpm;};

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
            goreleaser # Release builds (task release-test)
            cosign # Artifact signing/verification
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
