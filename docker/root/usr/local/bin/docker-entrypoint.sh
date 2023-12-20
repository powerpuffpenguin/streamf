#!/bin/sh
set -e
if [ "$1" == "default-command" ];then
    exec streamf -conf /data/streamf.jsonnet
else
    exec "$@"
fi