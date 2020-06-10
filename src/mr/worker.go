package mr

import "fmt"
import "log"
import "net/rpc"
import "hash/fnv"
import "os"
import "io/ioutil"
import "time"
import "sort"
import "math/rand"
import "encoding/json"
//import "../util"

//
// Map functions return a slice of KeyValue.
//
type KeyValue struct {
	Key   string
	Value string
}

//
type KeyValues struct {
	Key string
	Values []string
}

// for sorting by key.
type ByKey []KeyValue

// for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

//
// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
//
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}


//
// main/mrworker.go calls this function.
//
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {
	// Your worker implementation here.

	//randInit()
	//prefix := fmt.Sprintf("out-%s", RandStringRunes(8))

	taskFinishedArgs := TaskFinishedArgs{mapTask}
	taskFinishedReply := TaskFinishedReply{false}
	call("Master.TaskFinished", &taskFinishedArgs, &taskFinishedReply)
	for taskFinishedReply.TaskFinished != true {
		// check if all map are finished
		call("Master.TaskFinished", &taskFinishedArgs, &taskFinishedReply)

		// get map task
		getFileArgs := GetFileArgs{mapTask}
		getFileReply := GetFileReply{}
		call("Master.GetFile", &getFileArgs, &getFileReply)

		// do task
		// --如果返回文件为空，说明没有todo的文件，此时休眠1s再请求
		filename := getFileReply.FileName
		
		if filename == "" {
			//fmt.Println("nil")
			time.Sleep(time.Second)
			continue
		}
		//fmt.Println(filename, getFileReply.Id)
		intermediate := []KeyValue{}
		file, err := os.Open(filename)
		if err != nil {
			log.Fatalf("cannot open %v", filename)
		}
		content, err := ioutil.ReadAll(file)
		if err != nil {
			log.Fatalf("cannot read %v", filename)
		}
		file.Close()
		kva := mapf(filename, string(content))
		intermediate = append(intermediate, kva...)

		// --将kva写入对应的文件tmpFile
		sort.Sort(ByKey(intermediate))
		i := 0
		for i < len(intermediate) {
			j := i + 1
			for j < len(intermediate) && intermediate[j].Key == intermediate[i].Key {
				j++
			}
			values := []string{}
			for k := i; k < j; k++ {
				values = append(values, intermediate[k].Value)
			}
			key := intermediate[i].Key
			//intermediateName := fmt.Sprintf("%s-%d-%d", prefix, getFileReply.Id, ihash(key) % getFileReply.NReduce)
			intermediateName := fmt.Sprintf("out-%d-%d", getFileReply.Id, ihash(key) % getFileReply.NReduce)
			// If the file doesn't exist, create it, or append to the file
			outxy, err := os.OpenFile(intermediateName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Fatal(err)
			}
			kvs := KeyValues {
				Key: key,
				Values: values,
			}
			enc := json.NewEncoder(outxy)
			if err := enc.Encode(&kvs); err != nil {
				log.Fatal(err)
			}
			outxy.Close()
			i = j
		}

		// update map state
		setStateArgs := SetStateArgs{
			TaskType: mapTask,
			Id: getFileReply.Id,
			State: finished,
		}
		setStateReply := SetStateReply{false}
		call("Master.SetState", &setStateArgs, &setStateReply)
		if !setStateReply.Success {
			log.Fatal("map set state fail")
		}
		
	}

	// do reduce
	taskFinishedArgs = TaskFinishedArgs{rdcTask}
	taskFinishedReply = TaskFinishedReply{false}
	call("Master.TaskFinished", &taskFinishedArgs, &taskFinishedReply)
	for taskFinishedReply.TaskFinished != true {
		// check if all map are finished
		call("Master.TaskFinished", &taskFinishedArgs, &taskFinishedReply)

		// get reduce task
		getFileArgs := GetFileArgs{rdcTask}
		getFileReply := GetFileReply{}
		call("Master.GetFile", &getFileArgs, &getFileReply)
		
		// do task
		if getFileReply.Id >= getFileReply.NReduce {
			time.Sleep(time.Second)
			continue
		}
		//fmt.Println("reduce: ", getFileReply.Id)
		rdcId := getFileReply.Id
		oname := fmt.Sprintf("mr-out-%d", rdcId)
		ofile, _ := os.Create(oname)
		kva := make(map[string]([]string))
		for i := 0; i < getFileReply.NMap; i++ {
			//intermediateName := fmt.Sprintf("%s-%d-%d", prefix, i, rdcId)
			intermediateName := fmt.Sprintf("out-%d-%d",  i, rdcId)
			//fmt.Println(intermediateName)
			outxy, err := os.Open(intermediateName)
			if err != nil {
				//wd, _ := os.Getwd()
				//fmt.Println("wd: ", wd, " ", err)
			} 
			dec := json.NewDecoder(outxy)
			var kvs KeyValues
			for dec.More() {
				if err := dec.Decode(&kvs); err != nil {
					log.Fatal(err)
					break
				}

				for _, v := range kvs.Values {
					kva[kvs.Key] = append(kva[kvs.Key], v)
				}
			}
			outxy.Close()
			os.Remove(intermediateName)
		}
		keys := make([]string, 0, len(kva))
		for k := range kva {
			keys = append(keys, k);
			
		}
		sort.Strings(keys)
		for _, key := range keys {
			output := reducef(key, kva[key])
			// this is the correct format for each line of Reduce output.
			fmt.Fprintf(ofile, "%v %v\n", key, output)	
		}	
		ofile.Close()

		// update map state
		setStateArgs := SetStateArgs{
			TaskType: rdcTask,
			Id: getFileReply.Id,
			State: finished,
		}
		setStateReply := SetStateReply{}
		call("Master.SetState", &setStateArgs, &setStateReply)
		if !setStateReply.Success {
			log.Fatal("reduce set state fail")
		}
		
	}
	//fmt.Println("i am done")
	// uncomment to send the Example RPC to the master.
	// CallExample()

}

//
// example function to show how to make an RPC call to the master.
//
// the RPC argument and reply types are defined in rpc.go.
//
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	call("Master.Example", &args, &reply)

	// reply.Y should be 100.
	fmt.Printf("reply.Y %v\n", reply.Y)
}

//
// send an RPC request to the master, wait for the response.
// usually returns true.
// returns false if something goes wrong.
//
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := masterSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}

// Exists reports whether the named file or directory exists.
func Exists(name string) bool {
    if _, err := os.Stat(name); err != nil {
        if os.IsNotExist(err) {
            return false
        }
    }
    return true
}

func TestFileExists() {
	fmt.Println(Exists("rpc.go"))
	fmt.Println(Exists("../mr/rpc.go"))
	fmt.Println(Exists("wc.so"))
	fmt.Println(Exists("../main/wc.so"))
}

func randInit() {
    rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
    b := make([]rune, n)
    for i := range b {
        b[i] = letterRunes[rand.Intn(len(letterRunes))]
    }
    return string(b)
}