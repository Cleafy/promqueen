#!/usr/bin/env bash

while true
do
  logrotate /etc/logrotate.d/metrics
  sleep 300 # 5 minutes
done
