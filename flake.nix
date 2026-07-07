{
  description = "Go and Node.js environment with Svelte kit";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-26.05";
    flake-utils.url = "github:numtide/flake-utils";
    crow-ci.url = "https://codefloe.com/crowci/crowci-flake/archive/main.tar.gz";
  };

  outputs = { nixpkgs, flake-utils, crow-ci, ... }:
    # with flake-utils.lib; eachSystem allSystems (system:
    with flake-utils.lib; eachSystem ["x86_64-linux" "aarch64-darwin"] (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [ crow-ci.overlays.default ];
        };
      in {
        devShells.default = pkgs.mkShell {
          # TEMP WORKAROUND (do not commit): nix's nodejs_24 24.15.0 on
          # aarch64-darwin has broken worker_threads fd tracking, which spams
          # "File descriptor ... unmanaged mode" and can SIGKILL in CI.
          # See https://github.com/NixOS/nixpkgs/issues/525627.
          # Use Homebrew's node@24 (24.16.0, fixed) + corepack-provided pnpm
          # (pinned to pnpm@11.4.0 via package.json's packageManager field).
          buildInputs = [
            pkgs.go
            pkgs.gopls
            pkgs.crow-cli
            pkgs.mdbook
          ]
            ++ pkgs.lib.optionals (system == "aarch64-darwin") [ pkgs.pngpaste ]
            ++ pkgs.lib.optionals pkgs.stdenv.isLinux [ pkgs.wl-clipboard ];
          shellHook = ''
            export PATH="/opt/homebrew/opt/node@24/bin:$PATH"
            corepack enable >/dev/null 2>&1 || true
          '';
        };
      });
}

