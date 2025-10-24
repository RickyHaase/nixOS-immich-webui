This Doc needs to cover two things:

1. How to setup Cloudflare Tunnels
2. How to setup OAuth via Cloudflare Zero Trust

nixmich will handle all the server-side (immich and nix) setup, you just need to follow guides on how to setup the cloudflare side of things and then fill in the right fields in the nixmich web UI

## Cloudflare Tunnels
I'll sort this out some other time. On a time crunch tonight.

in local endpoint, it will be http://localhost:2283

NOTE: we will be using a SaaS app to protect the tunnel (kinda) via OAuth rather than a self-hosted app... I may look into seeing if the auth can be passed through on both to protect all routes but I doubt anything will come of it

BOT PRotection

## OAuth
more details to come. for now: https://github.com/immich-app/immich/discussions/8299
NOTE: you can use Cloudflare OTP as the identity... downside is that there is no name... setting up Google or others is a bit technical tho but it works nice

https://docs.immich.app/administration/oauth/
https://developers.cloudflare.com/cloudflare-one/access-controls/applications/http-apps/saas-apps/generic-oidc-saas/
