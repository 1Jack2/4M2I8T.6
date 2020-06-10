package mr

import "log"
import "net"
import "os"
import "net/rpc"
import "net/http"
import "sync"
import "time"

const (
	todo = 0
	doing = 1
	finished = 2
	mapTask = 0
	rdcTask = 1
)

type Master struct {
	// Your definitions here.
	Files []string
	MapStates []int
	RdcStates []int
	MapLock sync.Mutex
	NReduce int
	TaskTime []time.Time
}

// Your code here -- RPC handlers for the worker to call.
func (m *Master) GetFile(args *GetFileArgs, reply *GetFileReply) error {
	m.MapLock.Lock()
	defer m.MapLock.Unlock()
	if args.TaskType == mapTask {
		for i, v := range m.MapStates {
			if v == todo {
				m.MapStates[i] = doing
				m.TaskTime[i] = time.Now()
				reply.Id = i
				reply.FileName = m.Files[i]
				reply.NReduce = m.NReduce
				break
			}
		}
	} else {
		reply.Id = m.NReduce
		for i, v := range m.RdcStates {
			if v == todo {
				m.RdcStates[i] = doing
				m.TaskTime[i] = time.Now()
				reply.Id = i
				reply.NReduce = m.NReduce
				reply.NMap = len(m.Files)
				break
			}
		}
	}
	
	return nil
}

func (m *Master) SetState(args *SetStateArgs, reply *SetStateReply) error {
	m.MapLock.Lock()
	defer m.MapLock.Unlock()
	if args.TaskType == mapTask {
		m.MapStates[args.Id] = args.State
	} else {
		m.RdcStates[args.Id] = args.State
	}
	reply.Success = true	
	return nil
}

func (m *Master) TaskFinished(args *TaskFinishedArgs, reply *TaskFinishedReply) error {
	m.MapLock.Lock()
	defer m.MapLock.Unlock()
	if args.TaskType == mapTask {
		for _, v := range m.MapStates {
			if v == todo || v == doing {
				return nil;
			}
		}
	} else {
		for _, v := range m.RdcStates {
			if v == todo || v == doing {
				return nil;
			}
		}
	}
	reply.TaskFinished = true;	
	return nil
}
//
// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
//
func (m *Master) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}


//
// start a thread that listens for RPCs from worker.go
//
func (m *Master) server() {
	rpc.Register(m)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := masterSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

//
// main/mrmaster.go calls Done() periodically to find out
// if the entire job has finished.
//
func (m *Master) Done() bool {
	ret := true
	mapFinished := true
	// Your code here.
	m.MapLock.Lock()
	defer m.MapLock.Unlock()
	for _, v := range m.MapStates {
		if v == todo || v == doing {
			ret = false
			mapFinished = false
			break
		}
	}
	for _, v := range m.RdcStates {
		if v == todo || v == doing {
			ret = false
			break
		}
	}

	// check if task fail
	if !mapFinished {
		for i, v := range m.MapStates {
			if v == doing && time.Since(m.TaskTime[i]) > 10 * time.Second {
				m.MapStates[i] = todo
			}
		}
	} else {
		for i, v := range m.RdcStates {
			if v == doing && time.Since(m.TaskTime[i]) > 10 * time.Second {
				m.RdcStates[i] = todo
			}
		}
	}
	return ret
}

//
// create a Master.
// main/mrmaster.go calls this function.
// nReduce is the number of reduce tasks to use.
//
func MakeMaster(files []string, nReduce int) *Master {
	m := Master{
		Files: files,
		MapStates: make([]int, len(files)),
		RdcStates: make([]int, nReduce),
		NReduce: nReduce,
		TaskTime: make([]time.Time, max(len(files), nReduce)),
	}

	// Your code here.
	


	m.server()
	return &m
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}