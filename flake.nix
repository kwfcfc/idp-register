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
        devShells.default = pkgs.mkShellNoCC {
          # Node 24 matches the node:24-alpine CI/Dockerfile stage; pnpm_11
          # should match package.json's packageManager pin (CI gets that exact
          # version through corepack). The former Homebrew node@24 workaround
          # for NixOS/nixpkgs#525627 is gone: that issue was fixed upstream.
          buildInputs = [
            pkgs.go
            pkgs.gopls
            pkgs.nodejs_24
            pkgs.pnpm_11
            pkgs.crow-cli
            pkgs.go-jsonnet
            pkgs.mdbook
          ]
            ++ pkgs.lib.optionals (system == "aarch64-darwin") [ pkgs.pngpaste ]
            ++ pkgs.lib.optionals pkgs.stdenv.isLinux [ pkgs.wl-clipboard ];
        };
      });
}

