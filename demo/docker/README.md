# 使用docker快速部署
## 1. 在电脑上安装docker，并运行
## 2. 修改docker-compose.yaml配置文件
1. 创建本地数据库存储目录/opt/dockerdata/mongo 与配置文件mongo下面的volumes 前半部分对应。
2. 创建本地媒体服务存储目录 /opt/dockerdata/media/www 此文件用来存储zlmedia生成的视频文件，与配置文件sip和mediaserver下面的视频目录一致。
3. 将config.ini 复制到/opt/dockerdata/media/ 下面，如果针对zlmedia进行配置修改直接修改此文件。文件目录与mediaserver下面的配置文件目录一致。
4. 配置对外暴露的访问端口。sipserver的http API端口（比如8090），sipserver的udp信令交互端口（比如5060），mediaserver的http端口（比如8080），mediaserver的rpt端口（比如10000 udp/tcp)
5. 配置sipserver的通知地址，此地址是sip系统设备心跳注册等的通知地址。当sip接收到设备注册成功，心跳时会同步通知到此地址。
6. 配置播放地址 sipserver下的地址，此地址最终要转到mediaserver的http端口上去（比如8080），可以域名访问，比如域名http://xxxx 利用ningx直接将请求转发到mediaserver的http端口，也可以直接IP+端口号的访问，比如服务器ip地址:12.12.12.12 使用12.12.12.12:8080 也可以。
7. 配置rtp接受视频流的地址，此处必须设置ip+端口号样式，ip为录像机可以访问到的服务器ip地址，比如说部署mediaserver的服务器ip地址是12.12.12.12 部署的rtp端口为10000，则sipserver下的MEDIA_RTP：12.12.12.12:10000。
## 3. docker-compose 部署 ，服务启动
## 4. 使用
1. 调用sipserver 的httpapi接口生成对应的GB28181 参数，并将参数填写到录像机上。注意sip服务器地址就是部署服务的地址，比如12.12.12.12 sip服务器端口为sipserver的信令端口，比如5060.
2. 调用sipserver的播放地址，返回后根据地址进行播放。