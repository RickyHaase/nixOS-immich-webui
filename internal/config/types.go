package config

// ConfigVariables represents the JSON configuration structure that matches nixconfig.json
// Contains only the fields that were originally in the Go template for NixOS configuration
type ConfigVariables struct {
	System struct {
		TimeZone     string `json:"timeZone"`
		AutoUpgrade  bool   `json:"autoUpgrade"`
		UpgradeTime  string `json:"upgradeTime"`
		UpgradeLower string `json:"upgradeLower"`
		UpgradeUpper string `json:"upgradeUpper"`
	} `json:"system"`
	RemoteAccess struct {
		Tailscale struct {
			Enable  bool   `json:"enable"`
			AuthKey string `json:"authKey"`
		} `json:"tailscale"`
	} `json:"remoteAccess"`
}

// NixConfig contains all NixOS config settings for template compatibility
// Used only for HTML template rendering where the old structure is expected
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
	MLModel      string
}

// ToNixConfig converts ConfigVariables to NixConfig structure for template compatibility
func (cv *ConfigVariables) ToNixConfig() *NixConfig {
	nixConfig := &NixConfig{
		TimeZone:     cv.System.TimeZone,
		AutoUpgrade:  cv.System.AutoUpgrade,
		UpgradeTime:  cv.System.UpgradeTime,
		UpgradeLower: cv.System.UpgradeLower,
		UpgradeUpper: cv.System.UpgradeUpper,
		Tailscale:    cv.RemoteAccess.Tailscale.Enable,
		TSAuthkey:    cv.RemoteAccess.Tailscale.AuthKey,
		Email:        "",
		EmailPass:    false,
	}

	// Email fields and ML model are managed separately - get them from immich-config.json for template compatibility
	if immich, err := GetImmichConfig(); err == nil {
		nixConfig.Email = immich.Notifications.SMTP.Transport.Username
		nixConfig.EmailPass = immich.Notifications.SMTP.Transport.Password != ""
		nixConfig.MLModel = immich.MachineLearning.Clip.ModelName
	}

	return nixConfig
}

// ImmichConfig represents the Immich configuration JSON structure
type ImmichConfig struct {
	Backup          Backup          `json:"backup"`
	Notifications   Notifications   `json:"notifications"`
	Server          Server          `json:"server"`
	StorageTemplate StorageTemplate `json:"storageTemplate"`
	MachineLearning MachineLearning `json:"machineLearning"`
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

type MachineLearning struct {
	Enabled bool     `json:"enabled"`
	URLs    []string `json:"urls"`
	Clip    Clip     `json:"clip"`
}

type Clip struct {
	Enabled   bool   `json:"enabled"`
	ModelName string `json:"modelName"`
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
