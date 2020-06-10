package main

import (
	"../mr"
	//"../util"
	"encoding/json"
	//"io/ioutil"
	"log"
	"os"
	//"fmt"
)

func main() {
	//mr.TestFileExists()
	//util.FilePrint("out-0-0")
	testJSON()
}

func testJSON() {
	// create a tmpFile
	// tmpFile, err := ioutil.TempFile(".", "tmp-")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// encoding后写入tmpFile
	kvs := mr.KeyValues {
		Key: "test",
		Values: []string{"1", "2", "3", "4"},
	}
	
	
	// 将tmpFile的内容合并到out-x-y
	// If the file doesn't exist, create it, or append to the file
	//outxy, err := os.OpenFile("out-0", os.O_APPEND|os.O_CREATE, 0755)
	
	outxy, err := os.OpenFile("out-0", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	enc := json.NewEncoder(outxy)
	if err := enc.Encode(&kvs); err != nil {
		log.Fatal(err)
	}
	
	// for dec.More() {
	// 	var kva mr.KeyValues
	// 	if err := dec.Decode(&kva); err != nil {
	// 		fmt.Println("???")
	// 		log.Fatal(err)
	// 	}
	// 	fmt.Println("...")
	// 	fmt.Println(kva)
	// 	if err := enc.Encode(&kva); err != nil {
	// 		log.Fatal(err)
	// 	}
	// }
}