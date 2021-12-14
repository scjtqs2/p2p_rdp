# 使用说明

已初步可以使用。

1.需要用到一个具有公网ip的机器S 作为 发现服务器 eg `111.231.198.54:30124`

2.client侧分两端，一个被控侧（rdp服务端内网运行）。一个控制侧（想要远控的电脑运行）

3.分别运行两侧的client端。

ps: 使用上面的公共server发现端的话，只需要下载client的包就行了。
## 配置文件说明

1. 发现服务端 (运行名为server的包)

```yaml
host: ""  #目前没有使用，直接写死了0.0.0.0。后面再优化
port: 30124 #暴露出去的端口
```

2. 控制侧客户端 （运行名为client的包）

```yaml
serverhost: 111.231.198.54           #发现服务端的ip地址 支持域名
serverport: 30124             #发现服务端的暴露udp端口
type: client_client_type      #这个类型代表client侧
clientportfrosvc: 30124       #一个用来和发现服务端 通信的udp端口
clientportforp2ptrance: 30123 #用来p2p打洞的udp端口
rdpp2pport: 30122             #转发rdp的端口 eg:用远程桌面客户端 请求 127.0.0.1:30122
appname: rdp-p2p              #p2p打洞分组。需要和server侧客户端一致。可以当做一个简单的密码
remoterdpaddr: ""             #client侧不用管
```

3. 被控侧客户端 （运行名为client的包)

```yaml
serverhost: 111.231.198.54           #发现服务端的ip地址 支持域名
serverport: 30124              #发现服务端的暴露udp端口
type: client_server_type       #这个类型代表server侧
clientportfrosvc: 30124        #一个用来和发现服务端 通信的udp端口
clientportforp2ptrance: 30123  #用来p2p打洞的udp端口
rdpp2pport: 30122              #server侧不用理会
appname: rdp-p2p               #p2p打洞分组。需要和client侧客户端一致。可以当做一个简单的密码
remoterdpaddr: 192.168.50.80:3389 #需要控制的rdp服务端的地址
```

## 编译说明

1. 发现服务端

```shell
go build -o server ./server/cmd
#windows下
go build -o server.exe ./server/cmd
```

2. client侧/server侧客户端

```shell
go build -o client ./client/cmd
#windows下
go build -o client.exe ./client/cmd
```

## 后面有空再做优化