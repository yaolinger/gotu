# gonet

* go version 1.18

## 项目结构

* pkg：底层/中间层封装
  * xlog：日志库
  * xnet:：网络库，目前支持kcp，tcp
  * xmsg：数据包分割
  * xactor：actor模式
