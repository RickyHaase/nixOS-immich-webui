# This is the "big-daddy" single-file config. Using it for "alpha 1" testing and build before planned seperation

{ config, pkgs, ... }:

{
  imports =
    [ # Include the results of the hardware scan.
      ./hardware-configuration.nix
      # ./system.nix
      # ./zfs.nix
      # ./admin.nix # imported to make configurations to host external to webgui
      # ./networking.nix
      # ./immich.nix
      # ./remoteaccess.nix
    ];

  # Default Configurations generated by installation (minus hostname and timezone settings)
  # Honestly don't know what's necessary as I haven't tested changing anything...
  boot.loader.systemd-boot.enable = true;
  boot.loader.efi.canTouchEfiVariables = true;
  networking.networkmanager.enable = true;
  i18n.defaultLocale = "en_US.UTF-8";
  i18n.extraLocaleSettings = {
    LC_ADDRESS = "en_US.UTF-8";
    LC_IDENTIFICATION = "en_US.UTF-8";
    LC_MEASUREMENT = "en_US.UTF-8";
    LC_MONETARY = "en_US.UTF-8";
    LC_NAME = "en_US.UTF-8";
    LC_NUMERIC = "en_US.UTF-8";
    LC_PAPER = "en_US.UTF-8";
    LC_TELEPHONE = "en_US.UTF-8";
    LC_TIME = "en_US.UTF-8";
  };
  services.xserver.xkb = {
    layout = "us";
    variant = "";
  };
  users.users.testuser = {
    isNormalUser = true;
    description = "Test User";
    extraGroups = [ "networkmanager" "wheel" ];
    packages = with pkgs; [];
  };
  environment.systemPackages = with pkgs; [
  ];
  system.stateVersion = "24.11"; # Did you read the comment? TLDR; leave

  # ====== SYSTEM ======
  time.timeZone = "America/New_York";

  # #Enable Unattended Upgrades
  system.autoUpgrade.enable = true;
  system.autoUpgrade.dates = "02:00";
  system.autoUpgrade = {
    # flake = inputs.self.outPath;
    flags = [
    "--update-input"
    "nixpkgs"
    "-L" # print build logs
    ];
    randomizedDelaySec = "15min";
  };
  system.autoUpgrade.allowReboot = true;
  system.autoUpgrade.rebootWindow.lower = "02:30";
  system.autoUpgrade.rebootWindow.upper = "03:00";

  # ====== NETWORKING ======

  networking.hostName = "immich";
  # networking.domain = ".local";
  # networking.enableIPv6 = false;
  # boot.kernel.sysctl."net.ipv6.conf.all.disable_ipv6" = true;

  # Avahi (mDNS)
  services.avahi.enable = true;
  services.avahi = {
    openFirewall = true;
    hostName = "immich";
    publish = {
      #userServices = true;
      enable = true;
      #domain = true;
      addresses = true;
      #workstation = true;
    };
    extraServiceFiles = {
      immich = ''
        <service-group>
          <name replace-wildcards="yes">%h</name>
          <service>
           <type>_http._tcp</type>
           <port>80</port>
          </service>
        </service-group>
      '';
    };
  };

  # Caddy (reverse proxy)
  services.caddy.enable = true;
  services.caddy = {
    virtualHosts."immich.local:80".extraConfig = ''
      reverse_proxy http://localhost:2283
    '';
    virtualHosts.":8080".extraConfig = ''
      reverse_proxy http://localhost:8000
    '';
  };

  networking.firewall.allowPing = true;
  networking.firewall.allowedTCPPorts = [ 80 8080 ];

  # ====== ZFS ======
  networking.hostId = "12345678";

  boot.supportedFilesystems = [ "zfs" ];
  boot.zfs.forceImportRoot = false;
  boot.zfs.extraPools = [ "tank" ];

  services.zfs.autoScrub.enable = true;

  services.sanoid.enable = true;
  services.sanoid  = {
    interval = "hourly";
    datasets = {
      "tank" = {
        recursive = true;
        autoprune = true;
        autosnap = true;
        hourly = 24;
        daily = 7;
        weekly = 0;
        monthly = 0;
        yearly = 0;
      };
    };
  };

  # ====== IMMICH ======
  virtualisation.docker.enable = true;
  virtualisation.docker.autoPrune.enable = true;
  virtualisation.docker.autoPrune.dates = "monthly";

  environment.systemPackages = with pkgs; [
    wget
  ];

  # Testing with non-root user for immich. Still not fully-baked
  # users.groups.immich = {};
  # users.users.immich = {
  #   isSystemUser = true;
  #   group = "immich";
  #   description = "Immich";
  #   extraGroups = [ "docker" ];
  # };

  # expects docker-compose.yml and .env file for Immich to be stored in /root/immich-app
  # Didn't quite get the systemd service to check and pull this working but decided against it as I'll be managing this through templates contained within the Go binary
  systemd.services.immich-app = {
    description = "Manage Immich Compose Stack";
    requires = [ "docker.service" ];
    after = [ "docker.service" ];
    wantedBy = [ "multi-user.target" ];

    serviceConfig = {
      Type = "simple";
      ExecStart = "${pkgs.docker}/bin/docker compose up";
      ExecStop = "${pkgs.docker}/bin/docker compose down";
      Restart = "always";
      WorkingDirectory = "/root/immich-app";
      TimeoutStopSec = "90";
    };
  };

  # ====== REMOTE ACCESS ======
  services.tailscale.enable = false;

  # create a oneshot job to authenticate to Tailscale
  systemd.services.tailscale-autoconnect = {
    description = "Automatic connection to Tailscale";

    # make sure tailscale is running before trying to connect to tailscale
    after = [ "network-pre.target" "tailscale.service" ];
    wants = [ "network-pre.target" "tailscale.service" ];
    wantedBy = [ "multi-user.target" ];

    # set this service as a oneshot job
    serviceConfig.Type = "oneshot";

    # have the job run this shell script
    script = with pkgs; ''
    # wait for tailscaled to settle
    sleep 2

    # check if we are already authenticated to tailscale
    status="$(${tailscale}/bin/tailscale status -json | ${jq}/bin/jq -r .BackendState)"
    if [ $status = "Running" ]; then # if so, then do nothing
        exit 0
    fi

    # otherwise authenticate with tailscale
    ${tailscale}/bin/tailscale up -authkey tskey-auth-kV7bYL6CNTRL-GXXhAHWhHXAVTcumJyyxXAc2cyjxQ3QkD --ssh
    '';
    # Maybe add ssh and/or serve options

  };
}
