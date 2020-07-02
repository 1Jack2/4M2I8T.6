package kvraft

import "../labrpc"
import "crypto/rand"
import "math/big"
import (
	"time"
	"sync"
)


type Clerk struct {
	servers []*labrpc.ClientEnd
	// You will have to modify this struct.

	clerkId		int
	comandId	int
	preLeaderId int
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

func MakeClerk(servers []*labrpc.ClientEnd) *Clerk {
	ck := new(Clerk)
	ck.servers = servers
	// You'll have to add code here.

	ck.clerkId = int(nrand())
	return ck
}

//
// fetch the current value for a key.
// returns "" if the key does not exist.
// keeps trying forever in the face of all other errors.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.Get", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
//
func (ck *Clerk) Get(key string) string {

	// You will have to modify this function.
	j := ck.preLeaderId
	args := GetArgs{
		Key:		key,
		ClerkId:	ck.clerkId,
		CommandId:	ck.comandId,
	}
	reply := GetReply{}
	var done bool
	var mu sync.Mutex
	for {
		mu.Lock()
		doneVal := done
		mu.Unlock()
		if doneVal {
			break
		}
		go func(done *bool, mu *sync.Mutex, j *int, args0 *GetArgs, reply0 *GetReply) {
			ok := false
			i := *j
			args := *args0
			reply := *reply0
			for !ok {
				ok = ck.servers[i].Call("KVServer.Get", &args, &reply)
				mu.Lock()
				doneVal := *done
				mu.Unlock()
				if doneVal {
					return
				}
				err := reply.Err
				if err == OK || err == ErrNoKey {
					break
				}
				ok = false
				i = (i + 1) % len(ck.servers)
				time.Sleep(10 * time.Millisecond)
			}
			mu.Lock()
			reply0.Value = reply.Value
			*j = i
			*done = true
			mu.Unlock()
		}(&done, &mu, &j, &args, &reply)
		DPrintf("create a new Get go routine")
		time.Sleep(500 * time.Millisecond)
	}
	mu.Lock()
	ck.preLeaderId = j
	mu.Unlock()
	ck.comandId++
	return reply.Value
}

//
// shared by Put and Append.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.PutAppend", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
//
func (ck *Clerk) PutAppend(key string, value string, op string) {
	// You will have to modify this function.
	j := ck.preLeaderId
	args := PutAppendArgs{
		Key:		key,
		Value:		value,
		Op:			op,
		ClerkId:	ck.clerkId,
		CommandId:	ck.comandId,  //是否需要加锁
	}
	reply := PutAppendReply{}

	var done bool
	var mu sync.Mutex
	for {
		mu.Lock()
		doneVal := done
		mu.Unlock()
		if doneVal {
			break
		}
		go func(done *bool, mu *sync.Mutex, j *int, args0 *PutAppendArgs, reply0 *PutAppendReply) {
			ok := false
			i := *j
			args := *args0
			reply := *reply0
			
			for !ok {
				// DPrintf("Call [%d] PutAppend", i)
				ok = ck.servers[i].Call("KVServer.PutAppend", &args, &reply)  //是否需要异步
				mu.Lock()
				doneVal := *done
				mu.Unlock()
				if doneVal {
					return
				}
				err := reply.Err
				if err == OK {
					break
				}
				ok = false
				i = (i + 1) % len(ck.servers)
				// DPrintf("prepare to Call [%d] PutAppend", i)
				time.Sleep(10 * time.Millisecond)
			}
			mu.Lock()
			*j = i
			*done = true
			mu.Unlock()
		}(&done, &mu, &j, &args, &reply)
		DPrintf("create a new PutAppend go routine")
		time.Sleep(500 * time.Millisecond)
	}
	mu.Lock()
	ck.preLeaderId = j
	mu.Unlock()
	ck.comandId++
}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}
func (ck *Clerk) Append(key string, value string) {
	ck.PutAppend(key, value, "Append")
}
