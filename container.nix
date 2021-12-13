{ config, pkgs, lib, ... }:

# This container is only used during development, so we don't bother securely
# configuring PostgreSQL here.

{
  boot.isContainer = true;

  services.postgresql = {
    enable = true;
    enableTCPIP = true;
    settings = {
      log_statement = "all";
      logging_collector = true;
    };
    authentication = pkgs.lib.mkOverride 10 ''
      local all all trust
      hostnossl all all 0.0.0.0/0 trust
    '';
    initialScript = pkgs.writeText "postgresql-init-script" ''
      CREATE ROLE log4shell WITH LOGIN PASSWORD 'log4shell' CREATEDB;
      CREATE DATABASE log4shell;
      GRANT ALL PRIVILEGES ON DATABASE log4shell TO log4shell;
    '';
  };

  networking = {
    useDHCP = false;
    firewall.allowedTCPPorts = [
      config.services.postgresql.port
    ];
  };
}
