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
            }(i)
        }
    }
    time.sleep()
}

```

## lab2B

Start() index 与 log index