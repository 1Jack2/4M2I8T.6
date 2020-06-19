package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import "sync"
import "sync/atomic"
import "../labrpc"
import (
	"time"
	"fmt"
	"math/rand"
)

// import "bytes"
// import "../labgob"

const (
	LEADER = 0
	FOLLOWER = 1
	CANDIDATE = 2
)

//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in Lab 3 you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh; at that point you can add fields to
// ApplyMsg, but set CommandValid to false for these other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int
}

// log entry
type LogEntry struct {
	Command interface{}
	Term int
}

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.
	applyCh chan ApplyMsg
	electionTimeout time.Duration
	lastHeartbeat time.Time
	state int
	currentTerm int
	votedFor int		//initial -1
	log []LogEntry
	commitIndex int
	lastApplied int
	nextIndex []int
	matchIndex []int
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	// DPrintf("call function GetState")
	var term int
	var isleader bool
	// Your code here (2A).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	term = rf.currentTerm
	isleader = rf.state == LEADER
	return term, isleader
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
}


//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}


type AppendEntriesArgs struct {
	Term int
	LeaderId int
	PrevLogIndex int
	PrevLogTerm int
	Entries []LogEntry
	LeaderCommit int

}

type AppendEntriesReply struct {
	Term int
	Success bool
}

func (rf * Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	

	if args.Term < rf.currentTerm {
		DPrintf("%d[%d] --> %d[%d] fail", args.LeaderId, args.Term,
				rf.me, rf.currentTerm)
		return
	}

	rf.lastHeartbeat = time.Now()
	//DPrintln("[", rf.me, "update HB ]", rf.lastHeartbeat)
	//DPrintf("[%d]<%d> update HB", rf.me, rf.currentTerm)
	rf.currentTerm = args.Term
	reply.Term = args.Term
	rf.state = FOLLOWER
	if len(rf.log) - 1 < args.PrevLogIndex {
		return
	}
	if args.PrevLogIndex != -1 && rf.log[args.PrevLogIndex].Term != args.Term {
		rf.log = rf.log[:args.PrevLogIndex]
		DPrintf("%d -> %d: reslice log", args.LeaderId, rf.me)
		return
	}
	for _, entry := range args.Entries {
		rf.log = append(rf.log, entry)
	}
	if args.LeaderCommit > rf.commitIndex {
		rf.commitIndex = func(x, y int) int {
			if (x > y) {
				return y
			}
			return x
		}(args.LeaderCommit, len(rf.log) - 1)
	}
	reply.Success = true
} 

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool{
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}
//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term int
	CandidateId int
	LastLogIndex int
	LastLogTerm int
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Term int
	VoteGranted bool
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if args.Term < rf.currentTerm {
		DPrintf("[%d]<%d> request vote [%d]<%d>: *term* fail", 
		args.CandidateId, args.Term, rf.me, rf.currentTerm)
		reply.Term = rf.currentTerm
		return
	}
	if rf.currentTerm == args.Term && rf.votedFor != -1 && rf.votedFor != args.CandidateId {
		DPrintf("[%d](cId: %d) request vote [%d](cId: %d): *CandidateId* fail", 
		args.CandidateId, args.CandidateId, rf.me, rf.votedFor)
		reply.Term = rf.currentTerm
		return
	}
	rf.currentTerm = args.Term
	reply.Term = args.Term
	// make sure up-to-date
	if len(rf.log) == 0 || args.LastLogTerm > rf.log[len(rf.log) - 1].Term ||
	args.LastLogTerm == rf.log[len(rf.log) - 1].Term && args.LastLogIndex >= len(rf.log) - 1 {
		reply.VoteGranted = true
		rf.votedFor = args.CandidateId
		rf.lastHeartbeat = time.Now()
		rf.state = FOLLOWER
	}
	DPrintf("[%d]<%d> vote for [%d]<%d>", rf.me, rf.currentTerm, args.CandidateId, args.Term)
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}


//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).

	rf.mu.Lock()
	defer rf.mu.Unlock()

	DPrintf("[%d]<%d> call func Start()", rf.me, rf.currentTerm)

	term = rf.currentTerm
	if rf.state == LEADER {
		isLeader = true
		index = len(rf.log)
		rf.log = append(rf.log, LogEntry{
									Command: command, 
									Term: rf.currentTerm,
								})
	}

	return index, term, isLeader
}

//
// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
//
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).
	rf.applyCh = applyCh
	rf.electionTimeout = time.Duration(500 + rand.Intn(1000)) * time.Millisecond 
	//rf.electionTimeout = time.Duration(500 + me * 500) * time.Millisecond
	DPrintln("server ", rf.me, "electionTimeout ", rf.electionTimeout)
	rf.lastHeartbeat = time.Now()
	rf.state = FOLLOWER
	rf.currentTerm = 0
	rf.votedFor = -1
	rf.commitIndex = -1
	rf.lastApplied = -1
	rf.nextIndex = make([]int, len(peers))
	rf.matchIndex = make([]int, len(peers))

	// election time out
	go checkElection(rf)

	// heartbeat
	go sendHeartbeat(rf)

	// apply
	go checkApply(rf)

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())


	return rf
}

func checkElection(rf *Raft) {
	// mu := sync.Mutex{}
	for !rf.killed() {
		// fmt.Println(time.Now(), rf.me, "check electionTimeout")
		mu := sync.Mutex{}
		grantedCnt := 1 // vote self
		rf.mu.Lock()
		mu.Lock()
		if rf.state != LEADER && time.Since(rf.lastHeartbeat) >= rf.electionTimeout {
			
			DPrintln(fmt.Sprintf("[%d] start election term<%d>", rf.me, rf.currentTerm + 1), 
					time.Since(rf.lastHeartbeat), "*******************", rf.electionTimeout)
			
			rf.state = CANDIDATE
			rf.currentTerm++
			rf.votedFor = rf.me
			rf.lastHeartbeat = time.Now()
			rf.electionTimeout = time.Duration(300 + rand.Intn(1000)) * time.Millisecond 
			mu.Unlock()
			rf.mu.Unlock()
			for i := len(rf.peers) - 1; i >= 0; i-- {
				if i == rf.me {
					continue
				}
				//DPrintf("create goroutine: [%d] asks [%d] for vote", rf.me, i)
				go func(i int) {
					//DPrintf("[%d]<%d> is acquiring a lock", rf.me, rf.currentTerm)
					rf.mu.Lock()
					//DPrintf("[%d]<%d> has got a lock", rf.me, rf.currentTerm)
					requestVoteArgs := &RequestVoteArgs{
						Term: rf.currentTerm,
						CandidateId: rf.me,
						LastLogIndex: len(rf.log) - 1,
					}
					if requestVoteArgs.LastLogIndex >= 0 {
						requestVoteArgs.LastLogTerm = rf.log[len(rf.log) - 1].Term
					}
					//DPrintf("[%d]<%d> is ready to send RequestVote", rf.me, rf.currentTerm)
					rf.mu.Unlock()
					requestVoteReply := &RequestVoteReply{}
					ok := rf.sendRequestVote(i, requestVoteArgs, requestVoteReply)
					if !ok {
						rf.mu.Lock()
						DPrintf("RPC fail: [%d]<%d> send request vote to [%d] fail", rf.me, rf.currentTerm, i)
						rf.mu.Unlock()
						return
					}
					if requestVoteReply.Term < requestVoteArgs.Term {
						DPrintf("[%d]: requestVoteReply.Term < rf.currentTerm", rf.me)
						return
					}
					
					mu.Lock()
					defer mu.Unlock()
					rf.mu.Lock()
					defer rf.mu.Unlock()
					
					if requestVoteReply.Term > rf.currentTerm {
						DPrintf("[%d]: requestVoteReply.Term > rf.currentTerm", rf.me)
						rf.currentTerm = requestVoteArgs.Term
						rf.state = FOLLOWER
					}
					//DPrintf("%d state %d", rf.me, rf.state)
					if rf.state == CANDIDATE && requestVoteReply.VoteGranted {
						grantedCnt++
						//DPrintf("grantedCnt: %d", grantedCnt)
					}
					if 2 * grantedCnt > len(rf.peers) {
						if rf.state != CANDIDATE {
							return
						}
						rf.state = LEADER
						// init nextIndex[] matchIndex[]
						for j := len(rf.peers) - 1; j >= 0; j-- {
							rf.nextIndex[j] = len(rf.log)
							rf.matchIndex[j] = 0
						}
						DPrintf("[%d]<%d> become LEADER", rf.me, rf.currentTerm)
					}
				}(i)	
			}
		} else {
			mu.Unlock()
			rf.mu.Unlock()
		}
		// DPrintf("[%d] election goroutine is sleeping", rf.me)
		time.Sleep(500 * time.Millisecond)
		// term, isLeader := rf.GetState()
		// DPrintf("term[%d] server[%d] isLeader[%t]", term, rf.me, isLeader)
	}
}

func sendHeartbeat(rf *Raft) {
	for !rf.killed() {
		

		rf.mu.Lock()
		
		if rf.state == LEADER {
			mu := sync.Mutex{}
			replCount := 1
			if len(rf.log) - 1 == rf.commitIndex {
				rf.log = append(rf.log, LogEntry{Term: rf.currentTerm})
			}
			// currentState := rf.state
			currentTerm := rf.currentTerm
			currentLogLen := len(rf.log)
			currentCommitIndex := rf.commitIndex
			rf.mu.Unlock()
			for i := len(rf.peers) - 1; i >= 0; i-- {
				if i == rf.me {
					continue
				}
				go func(i int) {
					ok := false
					
					for !rf.killed() && !ok {
						rf.mu.Lock()
						if rf.state != LEADER || currentTerm != rf.currentTerm {
							rf.mu.Unlock()
							return
						}
						appendEntriesArgs := &AppendEntriesArgs{
							Term: 				currentTerm,
							LeaderId: 			rf.me,
							PrevLogIndex: 		rf.nextIndex[i] - 1,
							LeaderCommit: 		currentCommitIndex,
						}
						if appendEntriesArgs.PrevLogIndex >= 0 {
							appendEntriesArgs.PrevLogTerm = rf.log[appendEntriesArgs.PrevLogIndex].Term
						}
						if rf.nextIndex[i] < currentLogLen {
							if rf.nextIndex[i] >= 0 {
								appendEntriesArgs.Entries = rf.log[rf.nextIndex[i] : currentLogLen]
							} else {
								appendEntriesArgs.Entries = rf.log[: currentLogLen]
							}
						}
						rf.mu.Unlock()

						appendEntriesReply := &AppendEntriesReply{}
						ok = rf.sendAppendEntries(i, appendEntriesArgs, appendEntriesReply)
						
						rf.mu.Lock()
						if rf.state != LEADER || currentTerm != rf.currentTerm {
							rf.mu.Unlock()
							return
						}
						if !ok {
							DPrintf("RPC fail: [%d] sent heartbeat to [%d] fail", rf.me, i)
							rf.mu.Unlock()
							continue
						}
						// ok == true
						if (!appendEntriesReply.Success) {
							if appendEntriesReply.Term > rf.currentTerm {
								rf.currentTerm = appendEntriesReply.Term
								rf.state = FOLLOWER
								rf.mu.Unlock()
								return
							}
							ok = false  //retry until success
							rf.nextIndex[i]--
							DPrintf("%d sent heartbeat to %d fails", rf.me, i);
						} else {
							rf.matchIndex[i] = appendEntriesArgs.PrevLogIndex + len(appendEntriesArgs.Entries)
							rf.nextIndex[i] = rf.matchIndex[i] + 1
							mu.Lock()
							replCount++
							nextCommitIndex := currentLogLen - 1
							if 2 * replCount > len(rf.peers) && rf.commitIndex < nextCommitIndex {
								rf.commitIndex = nextCommitIndex
							}
							mu.Unlock()
							// rf.matchIndex[i] = len(rf.log) - 1
							//DPrintf("[%d] --> [%d]", rf.me, i);
						}
						rf.mu.Unlock()
					}
				}(i)
			}
		} else {
			rf.mu.Unlock()	
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func checkApply(rf *Raft) {
	for !rf.killed() {
		rf.mu.Lock()
		if rf.lastApplied < rf.commitIndex {
			rf.lastApplied++
			commandIndex := rf.lastApplied
			command := rf.log[commandIndex].Command
			if command != nil {
				rf.applyCh <- ApplyMsg{
					Command:		rf.log[commandIndex].Command,
					CommandIndex:	commandIndex,
					CommandValid:	true,
				}
				DPrintf("[%d]<%d> applied log[%d]: nil ? %t", rf.me, rf.currentTerm, rf.lastApplied, rf.log[rf.lastApplied].Command == nil)
			}
		}
		rf.mu.Unlock()

		time.Sleep(10 * time.Millisecond)
	}
}