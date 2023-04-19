# gotu

* go version 1.18

## 项目结构

* pkg：底层/中间层封装
  * xlog：日志库
  * xnet:：网络库，目前支持tcp，udp，kcp，websocket
    * 网络层读写分离，未强制控制读写数据时序
  * xmsg：数据包分割
  * xactor：actor模式
  * xcommon：通用模块
  * xlatency：延迟模拟模块
  * xenv：环境变量读取
* script：脚本
  * build_all.sh：编译
  * build_docker.sh：docker 打包
  * tcp_test.sh：tcp 并发连接测试
* cmd：逻辑代码
  * udp_cli：测试udp client
  * udp_svr：测试udp server
  * tcp_cli：测试tcp client
  * tcp_svr：测试tcp server
  * [udp_tun](./README_udp_tun.md)：udp流量代理工具
