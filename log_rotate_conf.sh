#!/usr/bin/env bash

LOGFILE=$5

echo "LOG {
    ROTATION_PERIOD
    ROTATION_COUNT
    ROTATION_SIZE
    copytruncate
    delaycompress
    compress
    notifempty
    missingok
    sharedscripts
}" > "$LOGFILE"


sed -i "s+LOG+$1+" "$LOGFILE"
sed -i "s/ROTATION_PERIOD/$2/" "$LOGFILE"
sed -i "s/ROTATION_COUNT/rotate $3/" "$LOGFILE"

if [[ $4 =~ ^[0-9]+[kMG]$ ]]; then
  sed -i "s/ROTATION_SIZE/maxsize $4/" "$LOGFILE"

elif [[ $4 == -1 ]]; then
  sed -i '4d' "$LOGFILE"

else
  echo "rotation size $4 is not valid. Skipped configuration"
  sed -i '4d' "$LOGFILE"
fi