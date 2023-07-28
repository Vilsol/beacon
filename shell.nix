{ pkgs ? import <nixpkgs> {}}:

pkgs.mkShell {
  packages = [
    pkgs.chart-testing
    pkgs.helm-docs
  ];
}