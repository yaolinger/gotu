#! /bin/bash

# 暂时不考虑docker-compose

SCRIPT_PATH=$(cd "$(dirname $0)" && pwd)

CMD=$1

build() {
    sh $SCRIPT_PATH/build_docker.sh
}

clis=(
gotu_tcp_cli_0
gotu_tcp_cli_1
# gotu_tcp_cli_2
# gotu_tcp_cli_3
# gotu_tcp_cli_4
# gotu_tcp_cli_5
# gotu_tcp_cli_6
)

start() {
    docker run -d -p 5000:5000  --name gotu_tcp_svr gotu_tcp_svr /bin/bash
    for cli in "${clis[@]}"; do
        echo "docker run -d --sysctl net.ipv4.ip_local_port_range=\"15000 65000\" --net gotu0 --name $cli gotu_tcp_cli /bin/bash"
        docker run -d --sysctl net.ipv4.ip_local_port_range="15000 65000"  --name $cli gotu_tcp_cli /bin/bash
    done
}

stop() {
    docker stop gotu_tcp_svr
    for cli in "${clis[@]}"; do
        docker stop $cli
    done
}

clear() {
    docker rm gotu_tcp_svr
    for cli in "${clis[@]}"; do
        docker rm $cli
    done
}

log() {
    docker logs gotu_tcp_svr
    for cli in "${clis[@]}"; do
        docker logs $cli
    done
}


if [ "$CMD" == "start" ]; then
    stop
    clear
    start
elif [ "$CMD" == "stop" ]; then
    stop
elif [ "$CMD" == "clear" ]; then
    stop
    clear
elif [ "$CMD" == "build" ]; then
    build
elif [ "$CMD" == "log" ]; then
    log
else
    echo "usage: $0 [param]
                            : start  启动容器
                            : stop   关闭容器
                            : clear  清空容器
                            : log    查看日志
                            : build  构建镜像[tcp_svr/tcp_cli]"
fi
