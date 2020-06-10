package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import "os"
import "strconv"

//
// example to show how to declare the arguments
// and reply for an RPC.
//

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

// Add your RPC definitions here.
type GetFileArgs struct {
	TaskType int
}

type GetFileReply struct {
	Id int
	FileName string
	NMap int
	NReduce int
}

type SetStateArgs struct {
	TaskType int
	Id int
	State int
}

type SetStateReply struct {
	Success bool
}

type TaskFinishedArgs struct {
	TaskType int
}

type TaskFinishedReply struct {
	TaskFinished bool
}


// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the master.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func masterSock() string {
	s := "/var/tmp/824-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
