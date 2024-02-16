#!/bin/bash

SCRIPT_NAME=${0##*/}
SCRIPT_PATH="$( dirname -- "${BASH_SOURCE[0]}"; )/"

DEBUG="/dev/shm/${SCRIPT_NAME}.debug"
#DEBUG="/dev/null"

### CHANGE ME:
EMAIL_ADDRESSES="root@localhost"

echo "--- $(date) --- Param1 [ ${1} ] Param2 [ ${2} ]" > ${DEBUG}

### ALERT NAME RULES - "Plug Washing Machine" is sample alert name
if [ "${1}" == "Plug Washing Machine" ]
then
  if [ "${2}" == "firing" ]
  then
    echo "${1} alert is firing" | mail -s "${1}" ${EMAIL_ADDRESSES}
  else
    echo "${1} alert is resolved" | mail -s "${1}" ${EMAIL_ADDRESSES}
  fi
fi

