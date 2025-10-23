NOTE: just draft with notes for admins to retore in command line. there is currently no UI restore

1. setup server like it's a fresh server - DO NOT start Immich yet. Mount the backup drive and unzip the latest config
2. copy (replace existing one from fresh install) `variables.json` from latest backup into `/etc/nixos`
3. copy (replace existing one from fresh install) `immich-config.json` from backup into `/tank/immich`
4. copy (preferrably using rsync -a) the library folder from backups into `/tank/immich/library`
5. [official instructions from Immich](https://docs.immich.app/administration/backup-and-restore/#manual-backup-and-restore). They are modified below for the expected environment:
```
cd /tank/immich-config
docker compose down -v
rm -rf /tank/immich/pgdata/*
docker compose pull
docker compose create
docker start immich_postgres
sleep 10
gunzip --stdout "/path/to/backup/dump.sql.gz" \
| sed "s/SELECT pg_catalog.set_config('search_path', '', false);/SELECT pg_catalog.set_config('search_path', 'public, pg_catalog', true);/g" \
| docker exec -i immich_postgres psql --dbname=postgres --username=<DB_USERNAME>
docker compose up -d
```
