#!/usr/bin/env bash

CONF_LOG_FILE=/etc/logrotate.d/metrics

touch $CONF_LOG_FILE

./log_rotate_conf.sh "$ROTATION_FILE_LOG" "${ROTATION_PERIOD:=daily}" "${ROTATION_COUNT:=10}" "${ROTATION_SIZE:=-1}" $CONF_LOG_FILE

source ./rotate-metrics.sh &

# shellcheck disable=SC2086
/promqueen/promrec $PROM_ARGS