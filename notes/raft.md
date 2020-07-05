# Raft

## tips

- 理解论文中的相关内容
- 仔细阅读lab主页的要求与提示
- 仔细阅读课程助教的[guide](https://thesquareplanet.com/blog/students-guide-to-raft/)
- lec 2与lec 5关于Go的使用与如何debug

## lab2

过去这十天左右时间，我根据raft论文与6.824 lab2的提示“完成”了lab2 raft，简单的跑了一组测试，1/100的错误率，说明程序的确存在已知的bug。较低的错误率一方面说明bug不是很严重，另一方面也给debug带来了困难，意味着每次如果想验证debug的结果，就要花几十分钟跑一组测试（在我的pc上），如果还有未通过的测试，就只能根据log分析，继续debug，而如果全部测试都通过了，也不是意味着程序就没有bug了。对我而言，debug的成本太高了，等到一组测试跑完，思路都不连贯了。所以，暂时就这样吧。

![lab2-err](/notes/img/lab2-err.png)

总的来说，lab2最麻烦的一点是，为了保证效率，在进行RPC调用时，不能持有锁，所以在等待RPC调用返回的过程中可能有其他的goroutine改变了Raft的状态。由于test模拟了一个不可靠的网络环境，这种事件一定会发生。如果不进行充分的检查就根据reply更新server的状态，那么各种问题就会接踵而至。

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

### lab3A

Raft论文的figure 2给出了完成lab2的详细指导，实现的难度主要在于充分理解figure 2并根据test源码debug。相比之下，lab3A需要我们在client, kv application, Raft之间的交互方面有更多的思考，而debug的难度小于lab2。在lab3A debug的过程中，我发现之前我没有理解figure 8的含义，在lab2的test中这个问题没有充分暴露出来，而在lab3A中有专门针对figure 8的test，所以我才发现了这个问题，并根据test产生的日志在纸上一步步的画出了几个server的Raft追加log的过程，最后发现和figure 8本质上是一样的。重读了一遍相关内容后，有了更深入的理解。而debug的过程很轻松，只改了一行代码。然后就能稳定地通过lab 3A的全部测试。

![lab3-test](/notes/img/lab3-test.png)

看了lab3B部分的要求与提示后，我感觉实现思路应该不难理清，但是涉及到Raft的log的操作可能需要考虑额外corner case，故需要对Raft部分做出不小的改动。有点麻烦，等有时间了在做