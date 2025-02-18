# Environment Assumptions
These are the assumptions that are made about the environment in which the program will be run. For now, they are manually configured but a setup/installation script that configures all of these at install will need to be made.

1. There is a ZFS pool separate from the boot drive called "tank" with datasets tank/postgres and tank/immich which are where Immich data will be stored.
2. This program is stored in /root/ (until I find what best-practice is for storing a binary on NixOS that's not bundled in the package manager - it looks like /opt is used by NixOS, not sure if I can put things there not used by the package manager).
  - Might end up putting everything into /tank/config dataset and then the pool contains ALL config, data, and rollback data. All that the boot drive is good for is running the OS and apps (other than the webui binary). This would allow for a system to be rebuild on any hardware by just installing nix on the host machine, importing the zfs pool, and running the binary with a "restore" flag of some sort. It also means that if we lose the pool , we lose the system... maybe not do this
3. There has been no other configuration done on the system after installing it via the normal GUI installer.
