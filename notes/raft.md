# Raft

## lab2

TODO:

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

``` C
/** 6.824 2020 Midterm Exam *Raft*

*** Raft (1)

Ben Bitdiddle is working on Lab 2C. He sees that Figure 2 in the Raft
paper requires that each peer remember currentTerm as persistent
state. He thinks that storing currentTerm persistently is not
necessary. Instead, Ben modifies his Raft implementation so that when
a peer restarts, it first loads its log from persistent storage, and
then initializes currentTerm from the Term stored in the last log
entry. If the peer is starting for the very first time, Ben's code
initializes currentTerm to zero.

Ben is making a mistake. Describe a specific sequence of events in
which Ben's change would lead to different peers committing different
commands at the same index.

Answer: Peer P1's last log entry has term 10. P1 receives a
VoteRequest for term 11 from peer P2, and answers "yes". Then P1
crashes, and restarts. P1 initializes currentTerm from the term in its
last log entry, which is 10. Now P1 receives a VoteRequest from peer
P3 for term 11. P1 will vote for P3 for term 11 even though it
previously voted for P2.

*** Raft (2)

Bob starts with a correct Raft implementation that follows the paper's
Figure 2. However, he changes the processing of AppendEntries RPCs so
that, instead of looking for conflicting entries, his code simply
overwrites the local log with the received entries. That is, in the
receiver implementation for AppendEntries in Figure 2, he effectively
replaces step 3 with "3. Delete entries after prevLogIndex from the
log." In Go, this would look like:

rf.log = rf.log\[0:args.PrevLogIndex+1\]
rf.log = append(rf.log, args.Entries...)

Bob finds that because of this change, his Raft peers sometimes
commit different commands at the same log index. Describe a specific
sequence of events in which Bob's change causes different peers to
commit different commands at the same log index.

Answer:

(0) Assume three peers, 1 2 and 3.
(1) Server 1 is leader on term 1
(2) Server 1 writes "A" at index 1 on term 1.
(3) Server 1 sends AppendEntries RPC to Server 2 with "A", but it is delayed.
(4) Server 1 writes "B" at index 2 on term 1.
(5) Server 2 acknowledges ["A", "B"] in its log
(6) Server 1 commits/applies both "A" and "B"
(7) The delayed AppendEntries arrives at Server 2, and 2 updates its log to ["A"]
(8) Server 3, which only got the first entry ["A"], requests vote on term 2
(9) Server 2 grants the vote since their logs are identical.
(10) Server 3 writes "C" at index 2 on term 2.
(11) Servers 2 and 3 commit/apply this, but it differs from what Server 2 committed.

*/
```

### lab3
