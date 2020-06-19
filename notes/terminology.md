# Terminology

- Linearizability

## Byzantine & Non-Byzantine

Q: The paper mentions that Raft works under all non-Byzantine
conditions. What are Byzantine conditions and why could they make Raft
fail?

A: "Non-Byzantine conditions" means that the servers are fail-stop:
they either follow the Raft protocol correctly, or they halt. For
example, most power failures are non-Byzantine because they cause
computers to simply stop executing instructions; if a power failure
occurs, Raft may stop operating, but it won't send incorrect results
to clients.

Byzantine failure refers to situations in which some computers execute
incorrectly, because of bugs or because someone malicious is
controlling the computers. If a failure like this occurs, Raft may
send incorrect results to clients.

Most of 6.824 is about tolerating non-Byzantine faults. Correct
operation despite Byzantine faults is more difficult; we'll touch on
this topic at the end of the term.

## Linearizability

Q: How does linearizability differ from serializability?

A: The usual definition of serializability is much like
linearizability, but without the requirement that operations respect
real-time ordering. Have a look at this explanation:
[linearizability-versus-serializability](http://www.bailis.org/blog/linearizability-versus-serializability/)

Section 2.3 of the ZooKeeper paper uses "serializable" to indicate
that the system behaves as if writes (from all clients combined) were
executed one by one in some order. The "FIFO client order" property
means that reads occur at specific points in the order of writes, and
that a given client's successive reads never move backwards in that
order. One thing that's going on here is that the guarantees for
writes and reads are different.