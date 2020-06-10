package util

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"io"
	//"path/filepath"	
)

func FilePrint(path string) {
	ba ,err := readFile(path)
	if err != nil {
		fmt.Println("Error: %s\n", err)
	}
	fmt.Printf("The content of '%s' : \n%s\n", path, ba)
}

func readFile(path string) ([]byte, error) {
	// parentPath, err := os.Getwd()
	// if err != nil {
	// 	return nil, err
	// }

	//pullPath := filepath.Join(parentPath, path)
	pullPath := path
	file, err := os.Open(pullPath)
	if err != nil {
		return nil, err
	}

	defer file.Close()
	return read(file)
}

func read(fd_r io.Reader) ([]byte, error) {
	br := bufio.NewReader(fd_r)
	var buf bytes.Buffer

	for {
		ba, isPrefix, err := br.ReadLine()

		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		buf.Write(ba)
		if !isPrefix {
			buf.WriteByte('\n')
		}

	}
	return buf.Bytes(), nil
}