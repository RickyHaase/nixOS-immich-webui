# nixOS-immich-webui
A web UI designed to manage a nixOS system with the sole purpose of running and maintaining immich on the host.

NOTE: If compiling as-is, this will not change any nixOS configs. This was intentional because I don't want anyone who may download this release in the future to accidentally overwrite their nix config.
To have this change the configuration.nix file stored in /etc/nixos/, you'll need to change the value on line 14 (see comment).
To apply changes and run a `nixos-rebuild switch`, lines 267-270 will need to be uncommented before compiling.

This repo currently houses the Proof of Concept where I learned about how to do the basic things necessary to build the project in Go. It currently does nothing useful. See the post on https://notes.rickyhaase.com for more details on this.

Also to note, as this is just a PoC, it was thrown together with different components from ChatGPT (also see https://notes.rickyhaase.com to see my thoughts on this (eventually)). I doubt that any of this will remain as I learn more about Go and re-write things as I see best.
Things like error handling and config re-loading are all a mess and mostly non-functional, and the entire structure is likely going to change as this is my first project getting back into things and my first time learning a compiled language (for however much that may change things).
