#!/bin/sh
EXIT_CODE=0

EXEC_COMMAND_ARGS=$@

if test -f "$1"; then
    echo "$1 exists."
else
    echo "$1 not exists."
    EXEC_COMMAND_ARGS="/usr/bin/kollector "$@
fi

# exit code 2 means we have watch timeout just need to reconnect
while true; do
    $EXEC_COMMAND_ARGS
    EXIT_CODE=$?    
    echo "$EXEC_COMMAND_ARGS" " exited with code "$EXIT_CODE

    if [ $EXIT_CODE -ne 4 ]; then
        exit $EXIT_CODE
    fi
done