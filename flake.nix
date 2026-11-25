{
  description = "Kit — shared server, interaction primitives and tokens of the bhr-tools web apps";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";

  outputs = { self, nixpkgs }:
    let
      systems = [ "x86_64-linux" "aarch64-linux" ];
      forAll = f: nixpkgs.lib.genAttrs systems (system: f nixpkgs.legacyPackages.${system});
    in
    {
      # NixOS module shared by the apps (nix/app-module.nix).
      lib.mkAppModule = import ./nix/app-module.nix;

      devShells = forAll (pkgs: {
        default = pkgs.mkShell { buildInputs = with pkgs; [ go gopls nodejs ]; };
      });

      # `nix flake check` runs the Go tests; JS tests: node --test "tests/js/*.test.mjs".
      checks = forAll (pkgs: {
        go = pkgs.buildGoModule {
          pname = "kit-tests";
          version = "0";
          src = ./.;
          vendorHash = null;
          doCheck = true;
          installPhase = "mkdir -p $out";
        };
      });
    };
}
