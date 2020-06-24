# Raft

## lab2A

when rpc reply's term < rf.currentTerm, drop the reply
每次发送rpc前保持当前的状态，当收到rpc reply后，根据reply进行下一步处理时，要校验当前状态与之前保存的状态是否一致，比如：

- 当进行AppendEntries RPC之前需要保存当前的oldTerm，收到reply后检查当前的term，如果oldterm < term 或者当前sever不再是leader, 说明在等待RPC reply的过程中，server的状态发生了改变，此时应该丢弃该reply，不做处理。

- 还有一个更隐蔽的bug： 

``` Go
for !rf.killed() {
    if rf.state == LEADER {
        for i, v := range rf.peers {
            if i == rf.me {
                continue
            }
            go func(i int) {
                sendAppendEntries(i, args, reply)
                // 根据reply做后续处理...
            }(i)
        }
    }
    time.sleep()
}
```

一般我们为每一个server创建一个goroutine（以下简称AE goroutine），周期性发送AppendEntries RPC（当处于leader状态时），实现heartbeat与log replication。故RPC调用不应该阻塞该AE goroutine，为每个RPC调用单独创建一个rpc goroutine，等到RPC调用返回时，根据论文Figure 2中的rules改变当前server的状态。比如改变nextIndex[]。  

但是由于网络环境的不可靠，

## lab2B

Start() index 与 log index