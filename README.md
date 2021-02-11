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
* 协议：google protobuf
  * 完善的编解码、跨语言特性
  * 但内存消耗较大，大量的反射也对CPU消耗较高，是目前项目中的性能瓶颈
* C/S结构，聊天服务器可横向扩展

## 如何扩展
* 在玩家与聊天服之间加入一组网关服
  * 网关服的负载均衡可以自己实现，生产实践中更多的是使用云服务提供的负载均衡器
* 全局唯一的UUID对象，可由redis或etcd实现一个新的发号器
  * redis
    * 发号器字段固定在一个节点上，不会因为redis分片而产生一致性的问题
    * 在极端情况下，redis发生主备切换的时间段内，无法保证发号器发出的ID不重复
  * etcd
    * 自带的强一致性非常适合做发号器
    * 限于本次项目的时间，暂未接入

## 关键算法
* 脏字过滤：
  * 通过Trie树来加载脏字库
  * 藉由DFA判断敏感词，并替换其中的敏感字段
* 并发模型  
  * 每个房间一个协程，处理该房间内成员的聊天消息、脏字过滤等任务
  * 每个玩家的读写任务在单独的协程中处理
  * 玩家姓名的过滤交由全局唯一的房间管理器完成  
  
## 性能数据
![image](https://github.com/Gorjess/cctest/blob/master/profile.png)

## TODO
* 单元测试(目前仅多开客户端进行了测试)
* 一对一私聊（协议已完成，还未实现回调函数）
* GM命令： /popular n

  
  
