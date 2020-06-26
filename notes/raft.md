# Raft

## lab2

过去这十天左右时间，我根据raft论文与6.824 lab2的建议“完成”了lab2 raft，简单的跑了一组测试，1/100的错误率，说明程序的确存在已知的bug。较低的错误率一方面说明bug不是很严重，另一方面也给debug带来了困难，意味着每次如果想验证debug的结果，就要花几十分钟跑一组测试（在我的pc上），如果还有未通过的测试，就只能根据log分析，继续debug，而如果全部测试都通过了，也不是意味着程序就没有bug了。对我而言，debug的成本太高了，等到一组测试跑完，思路都不连贯了。所以，暂时就这样吧。

![lab2-err](/notes/img/lab2-err.png)
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
