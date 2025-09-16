# Remote Access

This document outlines the available options for accessing your Immich server remotely.

## Tailscale (Recommended)

Tailscale provides secure, encrypted access to your Immich server from anywhere.

### Features
- **Serve**: HTTP(S) access over your tailnet
  - Uses hostname (default domain name vs full tailnet for HTTPS)
- **Sharing**: Up to two additional users can access your tailnet
  - **Passkeys**: Use as an account that's not associated with any other service
  - **Access Control**: Share only Immich with users who aren't tailnet owners
- **Funnel**: Public access (use with caution)

### Setup
Configuration is handled through the web interface under Remote Access settings.

## Cloudflare Tunnel

For users with a Cloudflare account and domain name.

### Requirements
- Cloudflare Account
- Domain Name
- Zero Trust setup
- Cloudflared daemon
- OIDC configuration (recommended for security)

### Security Note
When using Cloudflare Tunnel with public access, strongly consider implementing Zero Trust authentication to protect your data.
