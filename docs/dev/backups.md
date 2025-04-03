# Backup design constraints/considerations

1. We will NOT format any disks. No data will ever be destroyed or overwritten except for that data which is in [external drive]/immich-server-backups.
2. We will ONLY support exFAT formatted disks. This is so that user data backed up to that disk will be accessible by them no matter what system they are connected to.
  - This results in a tricky situation with the above requirement, but users will have to pre-format their disks as exFAT before backups will work. Instructions for this process will be included in documentation.
  - NTFS and APFS will not mount, maybe ext4 or ZFS could work since the drivers are there but they will not be tested.
  - I would like to eventually also have a ZFS-based backup solution but anyone who wants that can use the admin.nix file to configure syncoid.
3. Users are welcome to use the backup HDD for other things so long as driveroot/immich-server-backups is expected to be overwritten during system backups.

## Current State of Backups
- The webUI will display all exFAT partitions that are connected via USB as "eligible disks." It does not check for size, available space, or anything additional to know if a backup will work. Backups can be done alongside other data, the only data that will be overwritten is if there's existing data in /partitionRoot/immich-server-backup/library or immich-server-backup/config/config-yyyy-mm-dd.zip.
- There is no progress checking, progress state, previous backup logs, backup verification, scheduling, or anything else that makes a decent backup system/UX. All that is currently developed is the simple rsync of the library folder and the copying and zipping of all other important config/data files.

## Backup UX

1. Connect USB disk (pre-formatted exFAT).
2. Select disk from dropdown in admin webUI.
3. Click Backup.
4. Progress bar or modal of some sort needs to start and the backup button is unclickable.

## Backup Backend
1. USB disks are read and eligible disks data is returned to the UX.
2. When a backup request is submitted, it contains the drive identifier to be used alongside the request.
  - Need to validate this before proceeding.
3. A process is spawned to start the new backup.
  - The program needs to keep track that 1) there is a backup in progress and 2) the progress of the backup to report back to the UI when it checks in.
4. The backup process does the following:
  - Grabs the latest DB dump from Immich and moves it to /usbdisk/immich-server-backups/database (future improvement is to generate a new one at the time of backup request).
  - Creates copies of all config files and zips them together into a file called config-yyyy-mm-dd.zip stored at /usbdisk/immich-server-backups/config.
  - Performs an rsync of the Immich Library folder to /usbdisk/immich-server-backups/library.
5. When done, sends an email or notification to the webUI and unmounts the disk.

## MISC notes during dev

### Future functionality considerations
1. Quick Backup - Just does an rsync that will run a diff based on metadata, deleting anything that’s been deleted from the source.
2. Verification - Verifies hashes of all files on both sides. Increases reads from both drives but does not increase writes.
3. Full - performs a full re-copy, increasing writes on target device. Runs hash verification afterwards.

Eventually will need to do some capacity checks:
1. Is the disk partition greater than or equal to the size of the library + 2GB (DB & config backups)?
2. Is the free space + existing Immich backups greater than the size of the library + DB dump?

At some point will need to start generating new pg-dumps but at this stage in development, we’re sticking with copying the one from the previous night.

SD Cards and internal disks might become an eligible backup medium in the future - for now, USB is the expected and only supported method.

### Backup History
Need to keep a log of backups.

Backup Log
[time] [status] [notes]

Will parse and generate an HTML page that will highlight success/fail in an easily glanceable UI… perhaps just include a small 1-week log next to the backup UI on the main page (think uptime kuma |||||||).

I'm thinking the UX could be just a link at the bottom of the manual backup section that links to another page "backup history and schedule" and on that page have the UI for selecting the options and enabling the backup cron as well as viewing the backup logs in a nicely formatted HTML table.

### scripts/commands tested before implementing in Go
#### Get disk Information

Need this function to get all eligible disks and return an array of disk objects.

Eligible disk:
- Transport == USB
- FSType == exFAT

Disk Object:
```
struct EligibleDisk {
    var label string
    var size string
    var model string
    var name string
}
```

Shell command from ChatGPT that works great:
```
lsblk -J -o NAME,SIZE,FSTYPE,TRAN,MODEL,LABEL | jq -r '
  .blockdevices[] | select(.tran == "usb") as $disk |
  $disk.children[]? | select(.fstype == "exfat") |
  "\(.label // "No Label")" | \(.size) | \($disk.model) | \(.name)'
```

#### Config files and DB backup
copies the latest Immich DB dump
```
(cd /tank/immich/backups && cp "$(ls -t /tank/immich/backups/ | head -n 1)" ~/tempconfig/"$(ls -t /tank/immich/backups/ | head -n 1)")

/run/media/root/Backup Disk
```
copy the current immich-config.json
```
cp /tank/immich/immich-config.json ~/tempconfig/immich-config.json
```
copy nixos config folder
```
cp -r /etc/nixos ~/tempconfig/nixos
```
copy immich compose
```
cp -r ~/immich-app ~/tempconfig/immich-app
```
add readme
```
echo "For restore instructions, go to https://github.com/rickyhaase/nixos-immich-webui/docs/restore-from-backup" > ~/tempconfig/readme.txt
```
zip and add to usb disk
```
mkdir /run/media/root/Backup\ Disk/immich-server-backups/config
zip -r "/run/media/root/Backup Disk/immich-server-backups/config/config-$(date +\%Y-\%m-\%d).zip" ~/tempconfig
```
remove temp files
```
rm -rf ~/tempconfig/*
```

NOTE: this method only permits one backup per day to be saved - the zip file just gets overwritten by whatever the latest backup was… is this desired??

#### library backup
```
rsync -a --info=progress2 --delete /tank/immich/library /run/media/root/Backup\ Disk/immich-server-backups/
```
NOTE: need to figure out how to pass progress to webUI

#### Backup POST

- Receive post request with device name
- Run Get Disk Information
- Compare received device name against eligible disks
    - If no match, exit returning error for invalid request - disk DNE or is ineligible. Please use a USB disk with a partition formatted exFAT.
- mount the disk
- get the mount point
- check if [mountpoint]/immich-server-backup exists
- create it if not
- rsync Immich library to [mountpoint]/immich-server-backup/
- Unmount disk
- inform user disk can be disconnected

NOTE: This flow changes a decent bit during implementation

#### Backup Progress
This is going to be an interesting problem to solve that I currently haven’t the least idea of how to do it but I’m sure HTMX has some trick up their sleeve.

#### Scheduled Backups
Cron job that calls binary with flags —backup —[dev]
Or do I want separate backup scripts that can be set up specifically for the cron job so that it works even without the Go binary being on the machine??
