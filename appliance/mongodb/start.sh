#!/bin/bash

case $1 in
  mongodb)
    chown -R mongodb:mongodb /data
    chmod 0700 /data
    shift
    exec gosu mongodb \
      /bin/drycc-mongodb $*
    ;;
  api)
    shift
    exec /bin/drycc-mongodb-api $*
    ;;
  *)
    echo "Usage: $0 {mongodb|api}"
    exit 2
    ;;
esac
