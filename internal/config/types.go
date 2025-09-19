package config

// NixConfig contains all NixOS config settings that will be modifiable via this interface
type NixConfig struct {
	TimeZone     string
	AutoUpgrade  bool   // also applies to allowReboot
	UpgradeTime  string // start of 1-hour window, interruption should be minimal during that window
	UpgradeLower string // value derived from UpgradeTime+30min
	UpgradeUpper string // value derived from UpgradeTime+60min
	Tailscale    bool
	TSAuthkey    string
	Email        string
	EmailPass    bool
}

// ImmichConfig represents the Immich configuration JSON structure
type ImmichConfig struct {
	Backup          Backup          `json:"backup"`
	Notifications   Notifications   `json:"notifications"`
	Server          Server          `json:"server"`
	StorageTemplate StorageTemplate `json:"storageTemplate"`
}

// Backup configuration for Immich
type Backup struct {
	Database Database `json:"database"`
}

// Database backup configuration
type Database struct {
	CronExpression string `json:"cronExpression"`
	Enabled        bool   `json:"enabled"`
	KeepLastAmount int    `json:"keepLastAmount"`
}

// Notifications configuration for Immich
type Notifications struct {
	SMTP SMTP `json:"smtp"`
}

// SMTP configuration for notifications
type SMTP struct {
	Enabled   bool      `json:"enabled"`
	From      string    `json:"from"`
	ReplyTo   string    `json:"replyTo"`
	Transport Transport `json:"transport"`
}

// Transport configuration for SMTP
type Transport struct {
	Host       string `json:"host"`
	IgnoreCert bool   `json:"ignoreCert"`
	Password   string `json:"password"`
	Port       int16  `json:"port"`
	Username   string `json:"username"`
}

// Server configuration for Immich
type Server struct {
	ExternalDomain   string `json:"externalDomain"`
	LoginPageMessage string `json:"loginPageMessage"`
	PublicUsers      bool   `json:"publicUsers"`
}

// StorageTemplate configuration for Immich
type StorageTemplate struct {
	Enabled                 bool   `json:"enabled"`
	HashVerificationEnabled bool   `json:"hashVerificationEnabled"`
	Template                string `json:"template"`
}

// BlockDevice represents a storage device from lsblk output
type BlockDevice struct {
	Name      string        `json:"name"`
	Size      string        `json:"size"`
	FSType    string        `json:"fstype"`
	Transport string        `json:"tran"`
	Model     string        `json:"model"`
	Label     string        `json:"label"`
	Children  []BlockDevice `json:"children"`
}

// LSBLKOutput represents the complete lsblk JSON output
type LSBLKOutput struct {
	BlockDevices []BlockDevice `json:"blockdevices"`
}

// EligibleDisk represents a disk eligible for backup
type EligibleDisk struct {
	PartitionLabel string
	PartitionSize  string
	Model          string
	Identifier     string
}