# gotu

* go version 1.18

## 项目结构

* pkg：底层/中间层封装
  * xlog：日志库
  * xnet:：网络库，目前支持tcp，udp，kcp，websocket
    * 网络层读写分离，未强制控制独写数据时序
  * xmsg：数据包分割
  * xactor：actor模式
  * xcommon: 通用模块
