//go:build dev

package config

// Development environment paths - used when building with -tags dev
const (
	NixDir            string = "test/nixos/"               // NixOS configuration directory (dev: test directory)
	ImmichDir         string = "test/tank/immich-compose/" // Immich docker-compose directory (dev: test directory)
	TankImmich        string = "test/tank/immich/"         // Immich config JSON location (dev: test directory)
	BackupHistoryFile string = "test/backup-history.json"  // Backup history log file (dev: test directory)
)
