# variables.nix - All user-configurable variables in consistent format
# This is the ONLY file that needs to be templated by Go
# All other .nix files are static and import these variables

{
  # System Configuration Variables
  timeZone = "{{.TimeZone}}";
  autoUpgrade = {{.AutoUpgrade}};
  upgradeTime = "{{.UpgradeTime}}";
  upgradeLower = "{{.UpgradeLower}}";
  upgradeUpper = "{{.UpgradeUpper}}";
  
  # Remote Access Variables
  tailscaleEnable = {{.Tailscale}};
  tsAuthKey = "{{.TSAuthkey}}";
  
  # Email Configuration Variables
  emailAddress = "{{.Email}}";
  emailPasswordSet = {{.EmailPass}};
  
  # Static Configuration (could be made configurable later)
  hostName = "immich";
  hostId = "12345678";
  zfsPoolName = "tank";
  immichWorkingDir = "/root/immich-app";
  immichTimeout = "90";
  adminPanelPort = 8080;
  webPublicPort = 80;
  immichInternalPort = 2283;
}

# Why this format works well:
# 1. Consistent structure - every variable follows same pattern
# 2. Easy to parse - predictable regex patterns
# 3. Easy to template - clear Go template variable placement  
# 4. NixOS native - pure Nix syntax, no foreign formats
# 5. Single responsibility - only contains data, no logic

# Parsing patterns (for Go):
# String: `variableName = "([^"]+)";`
# Boolean: `variableName = (true|false);`
# This is much more reliable than parsing scattered variables