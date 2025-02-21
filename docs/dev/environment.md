# Environment Assumptions
These are the assumptions made about the environment in which the program will be run. Currently, they are manually configured, but a setup/installation script that configures all of these during installation will need to be created.

1. There is a ZFS pool separate from the boot drive called "tank" with datasets tank/pgdata and tank/immich, where Immich data will be stored.
2. This binary is stored in /root/
3. No additional configuration has been done on the system after installing it via the normal GUI installer.
