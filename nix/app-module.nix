# NixOS module of a kit app: independent instances, each a hardened systemd
# service with its own port and data directory. Every app had the same
# module; its flake now writes
#
#   nixosModules.default = kit.lib.mkAppModule {
#     name = "getdone";                       # services.getdone, binary, user
#     title = "Get Done";
#     package = pkgs: self.packages.${pkgs.stdenv.hostPlatform.system}.default;
#     dbFile = "getdone.db";
#     # optional:
#     dataDir = name: if name == "default" || name == "perso" then "/var/lib/getdone" else "/var/lib/getdone-${name}";
#     extraOptions = lib: { accounts = lib.mkOption { … }; };
#     extraArgs = inst: [ "-accounts" inst.accounts ];
#     extraServiceConfig = inst: { LoadCredential = …; };
#   };
{ name
, title
, package
, dbFile ? "${name}.db"
, dataDir ? (instance: "/var/lib/${name}-${instance}")
, extraOptions ? (lib: { })
, extraArgs ? (inst: [ ])
, extraServiceConfig ? (inst: { })
}:
{ config, lib, pkgs, ... }:
let
  appName = name; # the instance submodule below has its own `name`
  cfg = config.services.${appName};
in
{
  options.services.${name} = {
    enable = lib.mkEnableOption "${title} instances";

    package = lib.mkOption {
      type = lib.types.package;
      default = package pkgs;
      description = "The ${title} package to use.";
    };

    instances = lib.mkOption {
      type = lib.types.attrsOf (lib.types.submodule ({ name, config, ... }: {
        # name: the instance's.
        options = {
          port = lib.mkOption {
            type = lib.types.port;
            description = "HTTP port of this instance.";
          };
          host = lib.mkOption {
            type = lib.types.str;
            default = "127.0.0.1";
            description = "Address the instance listens on.";
          };
          dataDir = lib.mkOption {
            type = lib.types.path;
            default = dataDir name;
            description = "Data directory of the instance (database, files).";
          };
          dbPath = lib.mkOption {
            type = lib.types.path;
            default = "${config.dataDir}/${dbFile}";
            description = "SQLite database of the instance.";
          };
          noAuth = lib.mkOption {
            type = lib.types.bool;
            default = false;
            description = "Disable the master password.";
          };
          user = lib.mkOption {
            type = lib.types.str;
            default = appName;
            description = "System user running the instance.";
          };
          group = lib.mkOption {
            type = lib.types.str;
            default = appName;
            description = "System group running the instance.";
          };
          openFirewall = lib.mkOption {
            type = lib.types.bool;
            default = false;
            description = "Open the port in the NixOS firewall.";
          };
        } // extraOptions lib;
      }));
      default = { };
      description = "Independent ${title} instances.";
    };
  };

  config = lib.mkIf (cfg.enable && cfg.instances != { }) {
    networking.firewall.allowedTCPPorts = lib.concatLists (
      lib.mapAttrsToList (_: inst: lib.optional inst.openFirewall inst.port) cfg.instances
    );

    users.users = lib.mkIf (lib.any (inst: inst.user == name) (lib.attrValues cfg.instances)) {
      ${name} = {
        isSystemUser = true;
        group = name;
        home = "/var/lib/${name}";
        createHome = true;
        description = "${title} daemon user";
      };
    };

    users.groups = lib.mkIf (lib.any (inst: inst.group == name) (lib.attrValues cfg.instances)) {
      ${name} = { };
    };

    systemd.services = lib.mapAttrs'
      (instance: inst:
        lib.nameValuePair (if instance == "default" then name else "${name}-${instance}") {
          description = "${title} (${instance})";
          after = [ "network.target" ];
          wantedBy = [ "multi-user.target" ];
          serviceConfig = {
            ExecStart = lib.escapeShellArgs ([
              "${cfg.package}/bin/${name}"
              "-host" inst.host
              "-port" (toString inst.port)
              "-data-dir" (toString inst.dataDir)
              "-db" (toString inst.dbPath)
            ] ++ lib.optional inst.noAuth "-no-auth" ++ extraArgs inst);
            Restart = "always";
            RestartSec = 3;
            User = inst.user;
            Group = inst.group;
            WorkingDirectory = toString inst.dataDir;
            ReadWritePaths = [ (toString inst.dataDir) ];
            # Hardening: the service only needs its data directory and the network.
            NoNewPrivileges = true;
            PrivateTmp = true;
            PrivateDevices = true;
            ProtectSystem = "strict";
            ProtectHome = true;
            ProtectKernelTunables = true;
            ProtectKernelModules = true;
            ProtectControlGroups = true;
            RestrictAddressFamilies = [ "AF_INET" "AF_INET6" "AF_UNIX" ];
            RestrictNamespaces = true;
            LockPersonality = true;
            MemoryDenyWriteExecute = true;
            SystemCallArchitectures = "native";
          } // lib.optionalAttrs (lib.hasPrefix "/var/lib/" (toString inst.dataDir)) {
            # systemd creates and owns the directory when it lives under /var/lib.
            StateDirectory = lib.removePrefix "/var/lib/" (toString inst.dataDir);
          } // extraServiceConfig inst;
        })
      cfg.instances;
  };
}
