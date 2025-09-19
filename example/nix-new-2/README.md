# Modular Nix Configuration with Direct Parsing

This directory demonstrates a practical approach that balances simplicity with organization, designed specifically for your existing workflow and requirements.

## Architecture Decision Analysis

### Why This Approach Over JSON?

After analyzing your needs for **parsing**, **generation**, and **rollback**, direct Nix parsing is better because:

**✅ Leverages Existing Code**
- Your `switchConfig()` and `applyChanges()` functions work unchanged
- Simple `.old` backup fits your established workflow  
- No need to rewrite working functionality

**✅ NixOS-Native Approach**
- Pure Nix ecosystem - no foreign JSON layer
- Users already expect to import Nix modules
- Fits the documented setup process in `configuration.nix.md`

**✅ Appropriate Complexity**
- 8 variables don't need enterprise-grade versioning
- Regex is reliable when you control the format
- Simpler architecture = fewer failure points

**✅ Modular Benefits Without Over-Engineering**
- Clear organization without complexity overhead
- Easy to enable/disable specific modules
- Matches your preference for separation

### JSON Approach Problems for Your Case:
❌ **Over-engineering**: Complex versioning system for simple needs  
❌ **Extra layer**: JSON → Nix conversion adds complexity  
❌ **Breaks workflow**: Would require rewriting working functions  
❌ **Foreign to NixOS**: Users expect `.nix` files, not JSON  

## File Structure & Responsibilities

```
/etc/nixos/
├── configuration.nix          # User's existing config + imports
├── variables.nix              # 🔧 TEMPLATED: All user settings
├── system.nix                 # 📄 STATIC: System configuration
├── networking.nix             # 📄 STATIC: Network/firewall/proxy
├── immich.nix                 # 📄 STATIC: Docker and Immich service
├── remoteaccess.nix           # 📄 STATIC: Tailscale configuration
├── zfs.nix                    # 📄 STATIC: ZFS and snapshots
└── admin.nix                  # 👤 USER: Additional packages/config
```

### Key Design Principles:

1. **Single Template**: Only `variables.nix` needs Go templating
2. **Static Modules**: All other `.nix` files are embedded in binary
3. **Consistent Format**: Reliable regex parsing on controlled structure
4. **Minimal User Impact**: Users add imports to existing `configuration.nix`
5. **Simple Backup**: `.old` files for `variables.nix` and `admin.nix`

## Parsing Reliability

### Problem with Current Approach:
Variables scattered throughout 212-line file with inconsistent formatting:
```nix
time.timeZone = "{{.TimeZone}}";                    # Line 49
system.autoUpgrade.enable = {{.AutoUpgrade}};       # Line 52  
services.tailscale.enable = {{.Tailscale}};         # Line 176
```

### Solution with Consistent Variables File:
```nix
# variables.nix - Predictable format
{
  timeZone = "America/New_York";
  autoUpgrade = true;
  tailscaleEnable = false;
  tsAuthKey = "tskey-auth-xyz";
}
```

**Parsing becomes bulletproof** because:
- You control both generation AND parsing
- Consistent format = reliable regex patterns
- Single file = no scattered variables to miss

## Integration with Existing Workflow

### Current Functions That Stay:
```go
// These work unchanged
func switchConfig() error { ... }    
func applyChanges() error { ... }
func CopyFile(src, dst string) error { ... }
```

### Functions to Replace:
```go
// Replace complex regex parsing
func loadCurrentConfig() → LoadCurrentConfig()

// Replace template system  
func saveTmpFile() → GenerateVariablesNix()
```

### Backup Strategy:
```go
// Simple, fits existing workflow
variables.nix → variables.nix.old
admin.nix → admin.nix.old
```

## User Setup Process

Maintains your documented approach from `configuration.nix.md`:

1. **User adds imports** to their existing `configuration.nix`:
```nix
imports = [
  ./hardware-configuration.nix
  # Add these:
  ./variables.nix
  ./system.nix
  ./networking.nix
  ./immich.nix
  ./remoteaccess.nix
  ./zfs.nix
  ./admin.nix
];
```

2. **Comment out hostname** (moved to `networking.nix`):
```nix
# networking.hostName = "nixos";
```

3. **Run setup** - your binary deploys all `.nix` files

## Implementation Benefits

### For Go Application:
- **Simpler parsing**: Predictable regex on consistent format
- **Easier generation**: String building vs complex templates
- **Better errors**: Clear parsing failures vs template issues
- **Less code**: Remove template embedding and parsing complexity

### For Users:
- **Familiar process**: Import Nix modules (established pattern)
- **Clear organization**: Each module has specific purpose
- **Easy customization**: Can modify `admin.nix` for additional packages
- **Reliable rollback**: Simple `.old` file restoration

### For Maintenance:
- **Fewer moving parts**: No JSON conversion layer
- **Clear separation**: Data (`variables.nix`) vs logic (modules)
- **Easy debugging**: Standard Nix error messages
- **Simple testing**: Can test modules independently

## Comparison: Current vs This Approach

| Aspect | Current | This Approach |
|--------|---------|---------------|
| **Template files** | 1 large file (212 lines) | 1 small file (variables only) |
| **Parsing complexity** | High (scattered variables) | Low (consistent format) |
| **Organization** | Mixed concerns | Clear module separation |
| **User setup** | Manual config editing | Import modules + comment hostname |
| **Backup strategy** | Single `.old` file | Per-file `.old` backups |
| **Maintainability** | Difficult (mixed template/config) | Easy (separated concerns) |

## Files in This Directory

### Core Implementation:
- **`variables.nix`** - Template with consistent format for reliable parsing
- **`system.nix`** - System configuration using imported variables  
- **`networking.nix`** - Network/firewall/proxy configuration
- **`immich.nix`** - Docker and Immich service configuration
- **`remoteaccess.nix`** - Tailscale remote access configuration
- **`zfs.nix`** - ZFS filesystem and snapshot management
- **`admin.nix`** - User-managed packages (copied from existing)

### Integration Examples:
- **`go-parsing-example.go`** - Complete implementation showing improved parsing
- **`configuration-imports-example.nix`** - What users add to their config

## Migration Strategy

1. **Phase 1**: Create new parsing functions for `variables.nix`
2. **Phase 2**: Replace template system with string generation  
3. **Phase 3**: Deploy modular `.nix` files with binary
4. **Phase 4**: Update user documentation for import process
5. **Phase 5**: Test configuration switching and rollback

## Why This is the Right Choice

This approach optimizes for your **actual requirements**:

- ✅ **Solves parsing**: Reliable regex on consistent format
- ✅ **Solves generation**: Simple string building
- ✅ **Solves rollback**: Simple `.old` file backup
- ✅ **Fits workflow**: Uses existing functions and user processes
- ✅ **Right complexity**: Appropriate for 8 variables and simple needs
- ✅ **Maintainable**: Clear separation without over-engineering
- ✅ **NixOS-native**: Pure Nix ecosystem approach

**Perfect for your use case**: Provides all the functionality you need while keeping the simplicity and reliability that works well for your project.