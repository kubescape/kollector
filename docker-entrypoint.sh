#!/bin/bash
EXIT_CODE=0

# exit code 2 means we have watch timeout just need to reconnect
while true; do
    $@
    EXIT_CODE=$?    
    echo "$@" " exited with code "$EXIT_CODE

    if [ $EXIT_CODE -ne 4 ]; then
        exit $EXIT_CODE
    fi
done