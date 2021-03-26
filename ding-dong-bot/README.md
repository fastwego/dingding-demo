# 如何在钉钉平台上 5 分钟内打造一个叮咚机器人

## 在钉钉注册一个机器人应用

- 配置名称、图标等基本信息
![](img/step-1-new-bot.png)

- 获取应用的 appkey/appsecret
![](img/step-2-bot-config.png)

- 配置机器人服务器 ip 白名单 和 回调 url
![](img/step-3-bot-url.png)

- 发布应用，点击调试后扫码进入钉钉测试群
![](img/step-4-bot-debug.png)

## 安装 fastwego/dingding 开发 sdk

`go get -u github.com/fastwego/dingding`

## 开发机器人

### 配置

- 将钉钉应用的配置更新到 `.env` 文件：
```.env

AppKey=xxxxxxxxxxx
AppSecret=xxxxxxxxxxxxxxxxxxx

LISTEN=:80
```

- 编写代码：

[main.go](./main.go)

## 编译 & 部署 到服务器

`GOOS=linux go build`

`chmod +x ./ding-dong-bot && ./ding-dong-bot`

## 测试群里发送消息

 将机器人加入到企业内部群里，@ding-dong-bot 发送 `ding` ，机器人就会回复 `dong`

![](img/demo.jpg)

## 结语

恭喜你！5分钟内就完成了一款钉钉机器人开发

完整演示代码：[https://github.com/fastwego/dingding-demo](https://github.com/fastwego/dingding-demo)