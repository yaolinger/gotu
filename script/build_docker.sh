#! /bin/bash

set -eo pipefail

SCRIPT_PATH=$(cd "$(dirname $0)" && pwd)

MOD_NAME=$(head -n1 $SCRIPT_PATH/../go.mod | cut -d" " -f2)

sh $SCRIPT_PATH/build_all.sh

BINS="cmd/*"

for BIN in $BINS; do
    BIN=$(basename $BIN)
    if [ "$BIN" == "tcp_svr" ] || [ "$BIN" == "tcp_cli" ]; then
        echo "cp $SCRIPT_PATH/../bin/$BIN  $SCRIPT_PATH/../docker/bin"
        cp $SCRIPT_PATH/../bin/$BIN  $SCRIPT_PATH/../docker/$BIN/bin

        pushd "$SCRIPT_PATH/../docker"
            docker build "$BIN" -t "gotu_$BIN"
        popd
    fi
done
