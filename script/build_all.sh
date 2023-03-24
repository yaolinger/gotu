#! /bin/bash

set -eo pipefail

SVR_NAME="$1"

SCRIPT_PATH=$(cd "$(dirname $0)" && pwd)

MOD_NAME=$(head -n1 $SCRIPT_PATH/../go.mod | cut -d" " -f2)

if [ "$SVR_NAME" == "" ]; then
    BINS="cmd/*"
else
    BINS="cmd/$SVR_NAME"
fi

for BIN in $BINS; do
    echo "- building $BIN ..."
    if [ -z "$GO_RACE" ]; then
        # 关闭cross-compiling, 允许在scratch docker镜像上运行
        # -race 和 CGO_ENABLED=0冲突, 开启CGO 要增加依赖库 libc6-compat
        CGO_ENABLED=0 GOOS=linux go build -o $SCRIPT_PATH/../bin/ $MOD_NAME/$BIN
    else
        GOOS=linux go build -race -o $SCRIPT_PATH/../docker/bin/ $MOD_NAME/$BIN
    fi
done

echo "SUCCESS."
