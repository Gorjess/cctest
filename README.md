# cctest
## 开始使用
### 环境配置
```bash
go mod tidy
```
### 服务端
```bash
cd serverimpl/chat
go build main.go
./main
```

### 客户端
切换到项目根目录
```bash
cd client
go build client.go
./client
```

## 设计思路


## 关键算法
* 脏字过滤：
  * 通过Trie树来加载脏字库
  * 藉由DFA判断敏感词，并替换其中的敏感字段
* 并发模型  
  * 每个房间一个协程，处理该房间内成员的聊天消息、脏字过滤等任务
  * 每个玩家的读写任务在单独的协程中处理
  * 玩家姓名的过滤交由全局唯一的房间管理器完成  
  
# 性能数据
https://github.com/Gorjess/cctest/blob/master/profile.png
  
  
