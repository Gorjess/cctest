# cctest
## 开始使用
### 服务端
```bash
make
sudo make install
sudo make run 
```
- 清理bin、日志等文件
```bash
sudo make clean
```

### 客户端
切换到项目根目录后
```bash
cd client
make
make run
```

## 设计思路
* 协议：google protobuf
  * 优点：完善的编解码、跨语言特性
  * 缺点：内存消耗较大，大量的反射对CPU消耗较高，是目前项目中的性能瓶颈
* C/S结构，聊天服务器可横向扩展
* 目前没有做网关服，但聊天服本身支持流量限制，可配置单位之间最大链接数量
  * serverimpl/chat/conf/config.json 中的 conn_num_per_second

## 如何扩展
* 在玩家与聊天服之间加入一组网关服
  * 网关服的负载均衡可以自己实现，但生产实践中更多的是使用云服务提供的负载均衡器
* 全局唯一的UUID对象，可由redis或etcd实现一个新的发号器
  * redis
    * 发号器字段固定在一个节点上，不会因为redis分片而产生一致性的问题
    * 在极端情况下，redis发生主备切换的时间段内，无法保证发号器发出的ID不重复
  * etcd
    * 自带的强一致性非常适合做发号器
    * 限于本次项目的时间，暂未接入

## 关键算法
* 历史消息：
  * 每个房间各有一个链表，用于读写历史消息
  * 超过50条后，将移除最早的一条消息（头节点）
* 脏字过滤：
  * 通过Trie来加载脏字库、判断输入的字符串并替换其中的敏感字符
* 并发模型  
  * 每个房间两个协程
    * 协程1：处理该房间内成员的聊天消息
    * 协程2：过滤并替换敏感词
  * 每个玩家的读写任务在单独的协程中处理
  * 玩家姓名的过滤交由全局唯一的房间管理器完成 
  
## 测试框架
[gotest](https://github.com/cweill/gotests)
* 目前已测试脏字过滤功能
  
## 性能数据
* 脏字替换性能：
```bash
BenchmarkFilter_Check-8   	    7498	    164053 ns/op
```
* 服务器吞吐：
![image](https://github.com/Gorjess/cctest/blob/master/profile.png)

## TODO
* 单元测试
    * 聊天功能，仅多开客户端进行了测试，需要分模块测试    
* GM命令： /popular n

  
  
