# Configuration Management Architecture

## Overview

The NixOS Immich WebUI uses a JSON-based configuration management system that leverages NixOS's native `builtins.fromJSON` functionality. This approach provides structured reliability for parsing current configuration state, generating new configurations, and handling rollbacks.

## Architecture Design

### Core Pattern: `builtins.fromJSON`

Every NixOS module uses the same consistent pattern:

```nix
{ config, pkgs, ... }:

let
  # Read JSON configuration using NixOS built-in functions
  vars = builtins.fromJSON (builtins.readFile ./nixconfig.json);
in
{
  # Use vars.section.setting throughout the configuration
  time.timeZone = vars.system.timeZone;
  services.tailscale.enable = vars.remoteAccess.tailscale.enable;
}
```

### Configuration File Structure

The `nixconfig.json` file contains all user-configurable settings in a structured format:

```json
{
  "system": {
    "timeZone": "America/New_York",
    "autoUpgrade": false,
    "upgradeTime": "02:00",
    "upgradeLower": "02:30", 
    "upgradeUpper": "03:00"
  },
  "remoteAccess": {
    "tailscale": {
      "enable": false,
      "authKey": "tskey-auth-placeholder"
    }
  }
}
```

## Benefits of This Approach

### 1. Bulletproof Parsing
- **No regex required**: JSON unmarshaling is standard and reliable
- **Type safety**: Structured data with clear validation
- **Error handling**: Standard JSON error messages for debugging

### 2. Simple Generation
- **Direct JSON marshaling**: `json.Marshal()` in Go replaces complex templates
- **No template files**: Eliminates embedded template complexity
- **Structured output**: Guaranteed valid JSON format

### 3. NixOS-Native Integration
- **Built-in functions**: Uses `builtins.fromJSON` and `builtins.readFile`
- **No external dependencies**: Standard NixOS functionality
- **Consistent pattern**: Same approach across all modules

### 4. Modular Organization
- **Separated concerns**: Each .nix file handles specific functionality
- **Consistent imports**: Same JSON reading pattern everywhere
- **Easy maintenance**: Clear separation of static vs dynamic configuration

## Configuration Modules

### Current Implementation

```
/etc/nixos/
├── nixconfig.json      # All user-configurable settings
├── system.nix          # System configuration (timezone, auto-upgrade)
├── remoteaccess.nix    # Tailscale VPN configuration  
├── networking.nix      # Network and proxy configuration
├── immich.nix          # Docker and Immich service configuration
├── zfs.nix            # ZFS storage configuration
├── admin.nix          # User-managed packages (backup only)
└── configuration.nix   # Main imports and base configuration
```

### Module Responsibilities

- **system.nix**: Timezone, automatic upgrades, USB backup support
- **remoteaccess.nix**: Tailscale service and authentication
- **networking.nix**: Hostname, Avahi mDNS, Caddy reverse proxy, firewall
- **immich.nix**: Docker service, Immich systemd service
- **zfs.nix**: ZFS pool configuration, snapshots, auto-scrub
- **admin.nix**: Advanced user packages (not managed by web UI)

## Go Integration

### Reading Current Configuration

```go
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

func LoadCurrentConfig() (*ConfigVariables, error) {
    data, err := os.ReadFile("nixconfig.json")
    if err != nil {
        return nil, err
    }
    
    var config ConfigVariables
    err = json.Unmarshal(data, &config)
    return &config, err
}
```

### Generating New Configuration

```go
func SaveConfig(config *ConfigVariables) error {
    // Create backup
    createBackup("nixconfig.json")
    
    // Generate JSON
    data, err := json.MarshalIndent(config, "", "  ")
    if err != nil {
        return err
    }
    
    // Write new configuration
    return os.WriteFile("nixconfig.json", data, 0644)
}
```

## Backup and Rollback Strategy

### Simple `.old` File Backup

```bash
# Before applying new configuration
nixconfig.json → nixconfig.json.old

# Rollback if needed
nixconfig.json.old → nixconfig.json
```

### Integration with Existing Workflow

The JSON approach integrates seamlessly with existing functions:

- `switchConfig()` - Works unchanged with JSON files
- `applyChanges()` - Works unchanged with JSON files  
- `CopyFile()` - Works unchanged for backup operations

## Comparison with Previous Approach

| Aspect | Go Templates | JSON with builtins.fromJSON |
|--------|-------------|----------------------------|
| **Parsing** | Regex patterns (brittle) | JSON unmarshaling (reliable) |
| **Generation** | Template execution | JSON marshaling |
| **Interface** | Template variables scattered | Structured JSON file |
| **Backup** | Multiple template files | Single JSON file |
| **Rollback** | Complex template restoration | Simple JSON file copy |
| **Debugging** | Template syntax errors | Standard JSON validation |
| **Maintenance** | Template + Go struct sync | Single source of truth |

## Development Workflow

### Adding New Configuration Options

1. **Update JSON structure** in `nixconfig.json`
2. **Update Go struct** to match JSON structure  
3. **Update relevant .nix module** to use new JSON field
4. **Update web form** to collect new setting
5. **Test configuration** with existing workflow

### Testing Configuration Changes

1. **Modify** `nixconfig.json` manually for testing
2. **Validate** JSON syntax: `nix-instantiate --eval -E 'builtins.fromJSON (builtins.readFile ./nixconfig.json)'`
3. **Test** module imports: `nixos-rebuild dry-build`
4. **Apply** changes: existing `switchConfig()` and `applyChanges()` functions

## Future Considerations

The current implementation focuses on the essential configuration variables extracted from the original Go template. The JSON structure can be extended as needed while maintaining the same consistent pattern across all modules.

The modular approach allows for:
- Easy addition of new configuration sections
- Individual module enable/disable functionality  
- Clear separation between static and dynamic configuration
- Maintained backward compatibility with existing deployment processes