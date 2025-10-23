//go:build !dev

package config

// Production environment paths - used by default (when not building with -tags dev)
const (
	NixDir            string = "/etc/nixos/"           // NixOS configuration directory (prod: system location)
	ImmichDir         string = "/tank/immich-compose/" // Immich docker-compose directory (prod: ZFS dataset)
	TankImmich        string = "/tank/immich/"         // Immich config JSON location (prod: ZFS dataset)
	BackupHistoryFile string = "./backup-history.json" // Backup history log file (prod: alongside binary)
)
