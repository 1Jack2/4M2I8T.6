package kvraft

import (
	"../labgob"
	"../labrpc"
	"log"
	"../raft"
	"sync"
	"sync/atomic"
	"time"
)

const (
	PUT = "Put"
	GET = "Get"
	APPEND = "Append"
)

const Debug = 1

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug > 0 {
		log.Printf(format, a...)
	}
	return
}


type Op struct {
	// Your definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
	CommandType			string
	CommandKey			string
	CommandValue		string
	ClerkId				int
	CommandId			int
	IsDup				bool
	//PreCmdAllDone	int
}

func (op1 Op) equals(op2 Op) bool{
	return op1.ClerkId == op2.ClerkId &&
			op1.CommandId == op2.CommandId
}

type KVServer struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg
	dead    int32 // set by Kill()

	maxraftstate int // snapshot if log grows this big

	// Your definitions here.
	kv				map[string]string
	log				[]Op
	lastApplied		int
	clerkLatestCmd	map[int]int
}


func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	// Your code here.

	op := Op{
		CommandType:		GET,
		CommandKey:			args.Key,
		ClerkId:			args.ClerkId,
		CommandId:			args.CommandId,
	}
	index, _, isLeader := kv.rf.Start(op)
	if !isLeader {
		reply.Err = ErrWrongLeader
		return
	}
	for {
		kv.mu.Lock()
		if kv.lastApplied >= index {
			logOp := kv.log[index]
			if !op.equals(logOp) {
				reply.Err = ErrWrongLeader
				DPrintf("expected: %v\nactual: %v", op, logOp)
			} else {
				reply.Err = OK
				if !logOp.IsDup {
					reply.Value = logOp.CommandValue
				} else {
					DPrintf("GET dup")
					for _, v := range kv.log {
						if v.equals(logOp) {
							reply.Value = v.CommandValue
							break
						}
					}
				}
			}
			kv.mu.Unlock()
			return
		}
		kv.mu.Unlock()
		time.Sleep(50 * time.Millisecond)
	}
}

func (kv *KVServer) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.

	op := Op {
		CommandType:		args.Op,
		CommandKey:			args.Key,
		CommandValue:		args.Value,
		ClerkId:			args.ClerkId,
		CommandId:			args.CommandId,
	}
	index, _, isLeader := kv.rf.Start(op)
	if !isLeader {
		reply.Err = ErrWrongLeader
		return
	}
	// DPrintf("isLeader: %t, index: %d", isLeader, index)
	for {
		kv.mu.Lock()
		if kv.lastApplied >= index {
			logOp := kv.log[index]
			if !op.equals(logOp) {
				reply.Err = ErrWrongLeader
			} else {
				reply.Err = OK
			}
			kv.mu.Unlock()
			return
		}
		kv.mu.Unlock()
		time.Sleep(50 * time.Millisecond)
	}
}

//
// the tester calls Kill() when a KVServer instance won't
// be needed again. for your convenience, we supply
// code to set rf.dead (without needing a lock),
// and a killed() method to test rf.dead in
// long-running loops. you can also add your own
// code to Kill(). you're not required to do anything
// about this, but it may be convenient (for example)
// to suppress debug output from a Kill()ed instance.
//
func (kv *KVServer) Kill() {
	atomic.StoreInt32(&kv.dead, 1)
	kv.rf.Kill()
	// Your code here, if desired.
}

func (kv *KVServer) killed() bool {
	z := atomic.LoadInt32(&kv.dead)
	return z == 1
}

//
// servers[] contains the ports of the set of
// servers that will cooperate via Raft to
// form the fault-tolerant key/value service.
// me is the index of the current server in servers[].
// the k/v server should store snapshots through the underlying Raft
// implementation, which should call persister.SaveStateAndSnapshot() to
// atomically save the Raft state along with the snapshot.
// the k/v server should snapshot when Raft's saved state exceeds maxraftstate bytes,
// in order to allow Raft to garbage-collect its log. if maxraftstate is -1,
// you don't need to snapshot.
// StartKVServer() must return quickly, so it should start goroutines
// for any long-running work.
//
func StartKVServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int) *KVServer {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(Op{})

	kv := new(KVServer)
	kv.me = me
	kv.maxraftstate = maxraftstate

	// You may need initialization code here.

	kv.applyCh = make(chan raft.ApplyMsg)
	kv.rf = raft.Make(servers, me, persister, kv.applyCh)

	// You may need initialization code here.
	
	kv.log = []Op{Op{}}
	kv.kv = make(map[string]string)
	kv.clerkLatestCmd = make(map[int]int)

	go	checkApply(kv)

	return kv
}

func checkApply(kv *KVServer) {
	for applyMsg := range kv.applyCh {
		kv.mu.Lock()
		op := applyMsg.Command.(Op)

		//检查是否是重复请求
		var dup bool
		lastId, ok := kv.clerkLatestCmd[op.ClerkId]
		if !ok {
			kv.clerkLatestCmd[op.ClerkId] = op.CommandId
		} else if lastId >= op.CommandId {
			DPrintf("kv[%d] drop dup op: %v", kv.me , op)
			dup = true 
			op.IsDup = true
		} else {
			// DPrintf("kv[%d] update clerkLatestCmd[%d]: %d", kv.me , op.ClerkId, op.CommandId)
			kv.clerkLatestCmd[op.ClerkId] = op.CommandId
		}
		// if len(kv.log) - 1 < applyMsg.CommandIndex {
		// 	kv.log = append(kv.log, op)
		// }
		kv.log = append(kv.log, op)
		if !dup {
			if op.CommandType == GET {
				val, ok := kv.kv[op.CommandKey]
				if !ok {
					kv.log[applyMsg.CommandIndex].CommandValue = ""
				} else {
					kv.log[applyMsg.CommandIndex].CommandValue = val
				}
			} else if op.CommandType == PUT {
				kv.kv[op.CommandKey] = op.CommandValue
			} else { //APPEND
				_, ok := kv.kv[op.CommandKey]
				if !ok {
					kv.kv[op.CommandKey] = op.CommandValue
				} else {
					//DPrintf(kv.kv[op.CommandKey])
					kv.kv[op.CommandKey] += op.CommandValue
					//DPrintf(kv.kv[op.CommandKey])
				}
			}
		}
		DPrintf("kv[%d] applied op: %v", kv.me , kv.log[applyMsg.CommandIndex])
		kv.lastApplied++
		kv.mu.Unlock()
	}
}
