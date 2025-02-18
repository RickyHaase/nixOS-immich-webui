# Remote Access

## Cloudflare Tunnel
- Cloudflare Account
- Domain Name
- Zero Trust
- Cloudflared
- ODIC

## Tailscale
- Serve
  - HTTP(S)
  - hostname (default domain name vs full tailnet for https)
- Sharing (up to two additional users)
  - Passkeys (use as an account that's not associated with any other service and can be used to authenticate other devices)
  - Access Control (share only immich with users who aren't tailnet owner. Devices authenticated with passkey only able to access immich)
- Funnel
  - Public access
