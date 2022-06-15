# gosip
sipserver,GB28181,ZLMediaKit

# gosip
和 [ZLMediaKit](https://github.com/xia-chu/ZLMediaKit) 一起使用，zlm配置文件中的webhook请配置为此项目的restfulapi地址，否则部分功能无法使用。
zlm免编译docker镜像 [zlm docker image](https://hub.docker.com/repository/docker/panjjo/zlmediakit)
交流方式：请加QQ群-542509000，@bzfj

---
## 快速开始
运行demo下面对应执行文件 ./demo/gosip_linux
## 功能支持
- [X] 设备注册管理
- [X] 实时预览
- [X] 远程回放
- [X] 历史文件获取
- [X] 支持流管理(Mongo存储维护），服务重启不会丢失流或者出现失控流。
- [X] 支持异步通知
- [X] 视频留录制,文件自动清理
## 功能描述
### 设备管理
- 设备分类
  + 用户设备
    用户设备为NVR/DVR 或者 支持28181协议的摄像头
    用户设备下属多个通道设备
  + 通道设备
    通道设备为连接到NVR/DVR上的摄像头 或者 支持28181协议的摄像头
- 设备编号申请流程
  1. 使用密码调用/users接口为用户设备申请相关数据
  2. 将接口返回信息以及密码填写到对应的用户设备上（28181服务）
  3. 等待用户设备链接sip服务注册
  4. 注册成功后调用/users/:id/devices接口申请通道设备相关数据
  5. 将接口返回信息填写到对应的通道设备上（通过用户设备的28181服务页面填写）
  6. 等待用户设备上报通道设备信息
  7. 完成
### 直播/回播
- 直播操作流程
  1. 完成设备注册及上报
  2. 使用设备编号调用/devices/:id/play接口 
  3. 接口返回ssrc（播放ID）和对应播放地址
  4. 使用播放地址请求播放（需做sign验证，验证通过播放）
  5. 播放过程不能前进后退，不能暂停，不能出现进度条
  6. 播放完成后调用/play/:id/stop停止播放id=ssrc(一般不需要关闭,因为多个直播公用一个ssrc)
  7. 如视频播放申请后1分钟内没有播放，则自动注销此次申请
  8. 一个通道设备只能存在一个直播申请
- 回播操作流程
  1. 完成设备注册及上报
  2. 查询通道设备对应的历史文件
  3. 选择历史文件中包含的时间段进行视频播放
  4. 使用设备编号调用/devices/:id/replay 
  5. 接口返回ssrc（播放ID）和对应播放地址
  6. 使用播放地址请求播放（需做sign验证，验证通过播放）
  7. 播放过程不能前进后退，不能暂停，不能出现进度条
  8. 播放完成后调用/play/:id/stop停止播放id=ssrc，
  9. 如视频播放申请后1分钟内没有播放，则自动注销此次申请
### 历史文件
- 操作流程
  1. 使用设备id调用/devices/:id/files 
  2. 接口同步返回历史文件列表(可能存在超多时间段数据，同步返回最多等待10S，超时后返回10S内接收到的数据）
### 视频录制
- 操作流程
  1. 使用视频流的ssrc调用接口/play/:id/record 开始录制
  2. 使用ssrc调用接口/record/:id/stop 结束录制，同步返回录制结果
  3. 超过录制最长时间时 通过异步notify推送录制结果，notify method：records.stop
  4. 使用录制功能一定要将zlm的webhook的回掉地址配置为sip restfulapi地址，否则录制结果获取不到。
---
## 接口
### api
- 用户设备注册
  + 请求方式：GET
  + 请求路径：/users
  + 请求参数
    |参数名|类型|说明| 
    | -- |-- | -- |
    |name|string|设备名称|
    |pwd|string|密码|

  + 返回参数(Users)
    |参数名|类型|说明| 
    | -- |-- | -- |
    |deviceid|string|设备id|
    |region|string|设备所在域|
    |active|string|最后活跃时间|
    |regist|bool|是否注册|
    |pwd|string|密码|
    |sysinfo|obj{sysInfo}|sip服务器信息|


- 用户设备更新
  + 请求方式：GET
  + 请求路径：/users/:id/update
  + 请求参数
    |参数名|类型|说明| 
    | -- |-- | -- |
    |id|string|用户设备id|
    |name|string|设备名称|
    |pwd|string|密码|

  + 返回参数(Users)
    |参数名|类型|说明| 
    | -- |-- | -- |
    |deviceid|string|设备id|
    |region|string|设备所在域|
    |active|string|最后活跃时间|
    |regist|string|是否注册|
    |pwd|string|密码|
    |sysinfo|obj{sysInfo}|sip服务器信息|
    
- 用户设备删除（自动删除对应通道设备）
  + 请求方式：GET
  + 请求路径：/users/:id/delete
  + 请求参数
    |参数名|类型|说明| 
    | -- |-- | -- |
    | id|string|用户设备id|

  + 返回参数

- 通道设备注册
  + 请求方式：GET
  + 请求路径：/users/:id/devices
  + 请求参数
    |参数名|类型|说明| 
    | -- |-- | -- |
    |id|string|用户设备id|

  + 返回参数(Devices)
    |参数名|类型|说明| 
    | -- |-- | -- |
    |deviceid|string|设备id|
    |region|string|设备所在域|
    |active|string|最后活跃时间|
    |name|string|设备名称|
    |status|string|是否在线On/Off|
    
- 通道设备删除
  + 请求方式：GET
  + 请求路径：/devices/:id/delete
  + 请求参数
    |参数名|类型|说明| 
    | -- |-- | -- |
    | id|string|用户设备id|

  + 返回参数
  
  
- 视频直播
  + 请求方式：GET
  + 请求路径：/devices/:id/play
  + 请求参数
    |参数名|类型|说明| 
    | -- |-- | -- |
    |id|string|通道设备id|

  + 返回参数
    |参数名|类型|说明| 
    | -- |-- | -- |
    |deviceid|string|通道设备id|
    |http|string|播放地址|
    |rtmp|string|播放地址|
    |ssrc|string|播放流id|
    
- 视频回播
  + 请求方式：GET
  + 请求路径：/devices/:id/replay
  + 请求参数
    |参数名|类型|说明| 
    | -- |-- | -- |
    |id|string|通道设备id|
    |start|int|开始时间|
    |end|int|结束时间|

  + 返回参数
    |参数名|类型|说明| 
    | -- |-- | -- |
    |deviceid|string|通道设备id|
    |http|string|播放地址|
    |rtmp|string|播放地址|
    |ssrc|string|播放流id|
    
- 停止播放
  + 请求方式：GET
  + 请求路径：/play/:id/stop
  + 请求参数
    |参数名|类型|说明| 
    | -- |-- | -- |
    |id|string|播放时返回的ssrc|

  + 返回参数
    |参数名|类型|说明| 
    | -- |-- | -- |
    
- 历史文件
  + 请求方式：GET
  + 请求路径：/devices/:id/files
  + 请求参数
```
url = http://localhost/devices/xxxx/files //xxxx为设备通道ID
{
  "start":1234, // 开始时间，时间戳
  "end":12345   // 结束时间，时间戳
}
```

  + 返回参数
```
{
  "code": "0", // 状态码，0成功其余为失败
  "time":1234, // 请求时间戳
  "id": "abcd", // 请求唯一ID 
  "data":{
    "daynum": 2, // 天数
    "timenum": 3, //  时间段数
    "list": [
        {
            "date": "2020-06-16",  // 天
            "items": [ // 包含的时间段
                {
                    "start": 1592524800,
                    "end": 1592561172
                },
                {
                    "start": 1592438400,
                    "end": 1592524800
                }
            ]
        },
        {
            "date": "2020-06-17",
            "items": [
                {
                    "start": 1592436932,
                    "end": 1592438400
                }
            ]
        }
    ]
  }
}
```


- 开始录制接口
  + 请求方式：GET
  + 请求路径：/play/:id/record
  + 请求参数
```
url = http://localhost/play/xxxx/record //xxxx为流ssrc
{
  
}
```

  + 返回参数
```
{
    "code": "0",
    "data": "pQg0wZvXucIt0PVGkn9yNSh0kwvHb4gP",
    "time": 1618043462,
    "id": "MCpWLu9D4bp3iQ5g"
}
```


- 停止录制接口
  + 请求方式：GET
  + 请求路径：/record/:id/stop
  + 请求参数
```
url = http://localhost/record/xxxx/stop //xxxx为流ssrc
{
  
}
```

  + 返回参数
```
{
    "code": "0",
    "data": "http://127.0.0.1:18081/record/rtp/29B92711/2021-04-10/16-31-03.mp4", // 录制文件播放或下载地址
    "time": 1618043479,
    "id": "VjCfI1cns4TkB77J"
}
```


- 添加流代理
  + 请求方式：GET
  + 请求路径：/addproxy
  + 请求参数
```
url = http://localhost/addproxy 
{
  "url": "拉流的地址，支持rtsp/rtmp/hls等,必传",
  "tag": "流标识，一个流对应一个，如果存在重复的就会直接返回对应的播放地址，必传"
}
```

  + 返回参数
```
{
    "code": "0",
    "data": {
      "http":"参考播放接口",
      "rtmp":"",
      "ws-flv:""
    },
    "time": 1618043462,
    "id": "MCpWLu9D4bp3iQ5g"
}
```


- 删除流代理
  + 请求方式：GET
  + 请求路径：/delproxy
  + 请求参数
```
url = http://localhost/delproxy 
{
  "tag": "添加流代理时的tag"
}
```

  + 返回参数
```
{
    "code": "0",
    "data": "", 
    "time": 1618043479,
    "id": "VjCfI1cns4TkB77J"
}
```

### notify

- 用户注册通知
  + 请求方式：GET
  + 请求路径：配置文件配置
  + 请求参数
```
{
  "method": "users.regiester",
  "data": {
    Users // 请看用户NVR设备相关返回结构
  }
}
```

  + 返回参数
```
ok
```

- 用户活跃通知
  + 请求方式：GET
  + 请求路径：配置文件配置
  + 请求参数
```
{
  "method": "users.active",
  "data": {
    "deviceid": "xxxx", // 设备id
    "status": "ON", // 状态 
    "time": 17654621544 // 时间
  }
}
```

  + 返回参数
```
ok
```

- 通道活跃通知
  + 请求方式：GET
  + 请求路径：配置文件配置
  + 请求参数
```
{
  "method": "devices.active",
  "data": {
    "deviceid": "xxxx", // 设备id
    "status": "ON", // 状态 
    "time": 17654621544 // 时间
  }
}
```

  + 返回参数
```
ok
```


- 视频录制超时自动停止通知
  + 请求方式：GET
  + 请求路径：配置文件配置
  + 请求参数
```
{
  "method": "records.stop",
  "data": {
    "url": "http://xx/xx.mp4", //录制文件播放或下载地址
    "stream": "流ssrc",
    "app": "rtp"
  }
}
```

  + 返回参数
```
ok
```