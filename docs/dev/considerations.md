# Development Considerations

This document captures important implementation decisions and considerations for the project.

## Implementation Notes
- I am not planning to implement any authentication for the admin interface at this time. Authentication will be handled via whatever methods are offered through the proxy being used (Caddy basic auth for local access or for remote access, Cloudflare Zero Trust's auth mechanism).
  - Caddy basic auth password reset needs to be handled somehow... I am thinking either a recovery page on an unprotected path with an email password reset flow and/or a command-line flag that can be run if the user has physical access to the appliance.
- Not really sure if I need to manage or care about the root and user passwords on the system. The admin will choose those when installing NixOS on the metal, and I could reset those when taking over the config as root. I guess it's just something I haven't yet put much thought into and will need to make a decision at some point along the way (likely deployment script).
- HW Transcoding is a feature that is desired but is the lowest priority to implement.
- Limiting HW usage of the Immich containers could be something worth adding... would involve editing the compose file I think. When adding HW transcodes, this could be added as well. Might not be necessary because this is a dedicated Immich host but n-1 CPUs and n-0.5 GB RAM could be desired to keep other system operations responsive.
- [Need to run Immich as non-root](https://immich.app/docs/FAQ#how-can-i-run-immich-as-a-non-root-user) (and maybe protect against privilege escalation).
- Need to make a custom compose file for port exposure, transcodes, and rootless... how to manage against the prod compose, I'm not yet sure.

- Initially, I think I'll render everything server-side and just have simple load active config on page load. Later, I may implement HTMX and allow for dynamic loading of config from active (.nix), saved (.tmp), and previous (.old) configuration states.
  - I am definitely going to have to make use of HTMX... however, I might be able to get away with just making an SPA because the majority of this work is taken care of in the background.

- Where to store this program? It is currently stored in /root/ (until best practices for storing a binary on NixOS that is not bundled in the package manager are determined - it appears that /opt is used by NixOS, but it is unclear if it can be used for items not managed by the package manager).
  - Another option is to store everything in the /tank/config dataset, so the pool contains all configuration, data, and rollback data. This way, the boot drive is only used for running the OS and apps (other than the webui binary). This would allow the system to be rebuilt on any hardware by simply installing Nix on the host machine, importing the ZFS pool, and running the binary with a "restore" flag. However, this also means that if the pool is lost, the entire system is lost, so this approach may not be advisable.

## One-time setup activities

These should never have to happen after first setup:
- Installation of NixOS on hardware
- Storage Formatting - creation of ZFS pool and datasets
- Creation and import of Nix config files (and removal of hostname from configuration.nix)
