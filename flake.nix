{
  description = "Official Flake for Streamshower";

  inputs = {
    nixpkgs.url = "github:Nixos/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    {
      self,
      nixpkgs,
      flake-utils,
    }:
    let
      build-pkg =
        pkgs:
        pkgs.buildGoModule {
          pname = "streamshower";
          version = "0.1.0";
          src = ./.;
          vendorHash = "sha256-QM3zLWxVm54WriodBYy0fF0Iq/8wcM1RuORDP1zii8E=";
          meta = {
            description = "Go-based tui for streamserver";
            homepage = "https://github.com/HoppenR/streamshower";
            mainProgram = "streamshower";
          };
        };

      outputs = flake-utils.lib.eachDefaultSystem (
        system:
        let
          pkgs = import nixpkgs { inherit system; };
        in
        {
          packages = rec {
            streamshower = build-pkg pkgs;
            default = streamshower;
          };

          devShells.default = pkgs.mkShellNoCC {
            buildInputs = with pkgs; [
              go
              gofumpt
              gopls
            ];

            shellHook = /* bash */ ''
              export STREAMSHOWER_HOME=$(git rev-parse --show-toplevel) || exit
              export XDG_CONFIG_DIRS="$STREAMSHOWER_HOME/.nvim_config:$XDG_CONFIG_DIRS"
            '';
          };
        }
      );
    in
    outputs
    // {
      overlays.default = final: prev: {
        streamshower = self.packages.${final.stdenv.hostPlatform.system}.default;
      };
    };
}
