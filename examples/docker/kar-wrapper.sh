#!/bin/sh

if [ -z "$KAR_LOCAL_MODE" ]; then
    exec node "$MAIN"
else
    exec /kar/bin/kar -app $KAR_APP $KAR_EXTRA_ARGS node "$MAIN"
fi

