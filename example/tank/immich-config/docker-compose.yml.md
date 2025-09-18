# Immich Docker Configuration

This directory contains the Docker Compose configuration for running Immich. These files should be placed in `/root/immich-app/` on your target system.

## Setup Instructions

### 1. Copy Files to System

Copy the docker-compose.yml to your Immich application directory:
```bash
cp docker-compose.yml /root/immich-app/
```

### 2. Create Environment File

Create `/root/immich-app/.env` with your configuration:
```bash
# Database Configuration
DB_PASSWORD=postgres
DB_USERNAME=postgres
DB_DATABASE_NAME=immich

# Storage Locations (ZFS datasets)
UPLOAD_LOCATION=/tank/immich/library
DB_DATA_LOCATION=/tank/pgdata

# Immich Version
IMMICH_VERSION=release

# Optional: GPU acceleration
# CUDA_VISIBLE_DEVICES=all
```

### 3. Directory Structure

The configuration expects this directory structure:
```
/root/immich-app/
├── docker-compose.yml
├── .env
└── (optional hwaccel files)

/tank/immich/library/     # Photo storage (ZFS dataset)
/tank/pgdata/             # Database storage (ZFS dataset)
/tank/immich-config/      # Config backup location
```

### 4. Starting Immich

The Immich containers are managed by the systemd service defined in `immich.nix`:
```bash
# Start Immich
systemctl start immich-app

# Enable autostart
systemctl enable immich-app

# Check status
systemctl status immich-app
```

## Configuration Notes

- The docker-compose.yml uses environment variables from `.env`
- Storage locations map to ZFS datasets for data protection
- Database includes vector extension for ML features
- Redis provides caching for better performance
- Health checks ensure service reliability

## Updating Immich

To update Immich to the latest version:
1. Update `IMMICH_VERSION` in `.env` (or leave as `release` for latest)
2. Use the WebUI update function or restart the service:
   ```bash
   systemctl restart immich-app
   ```

For complete setup instructions, see the [Deployment Guide](../../docs/setup/deployment.md).
