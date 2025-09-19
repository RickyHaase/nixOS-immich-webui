# Configuration History

This directory contains versioned snapshots of configuration changes for rollback purposes.

## File Structure

- `variables-XXX.json` - Numbered configuration snapshots
- `current-version.txt` - Tracks the active version number
- `README.md` - This documentation

## Version History

### Version 001 (2024-09-15)
- Initial setup with minimal configuration
- Auto-upgrades disabled
- No email or Tailscale configured
- UTC timezone

### Version 002 (2024-09-16)  
- Enabled automatic system upgrades
- Configured email notifications
- Changed timezone to America/New_York
- Reduced snapshot retention

### Version 003 (2024-09-18) - Current
- Enabled Tailscale remote access
- Updated email configuration
- Current active configuration

## Rollback Process

To rollback to a previous version:

1. **Copy desired version**: `cp variables-002.json ../variables.json`
2. **Update version tracker**: `echo "002" > current-version.txt`
3. **Apply configuration**: Run `nixos-rebuild switch`

## Go Integration

The Go application can:

```go
// Read current version
version, _ := os.ReadFile("history/current-version.txt")

// Create new version
nextVersion := fmt.Sprintf("%03d", currentVersion + 1)
backupPath := fmt.Sprintf("history/variables-%s.json", nextVersion)

// Save current as backup before making changes
copyFile("variables.json", backupPath)

// Generate new configuration
newConfig := generateConfig(userSettings)
writeFile("variables.json", newConfig)

// Update version tracker
writeFile("history/current-version.txt", nextVersion)
```

## Benefits

- **Easy rollback**: Just copy an old JSON file
- **Version tracking**: Clear history of all changes
- **Diff capability**: Can compare any two versions
- **Atomic changes**: Each configuration change is a complete snapshot
- **No complex parsing**: Simple file operations