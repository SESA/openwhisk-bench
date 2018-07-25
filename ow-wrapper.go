package main

import (
	"fmt"
	"os"
	"os/exec"
	"bytes"
	"strings"
)

func main() {
	var buffer bytes.Buffer
	
	for i := 1; i < len(os.Args); i++ {
		buffer.WriteString(os.Args[i])
		buffer.WriteString(" ")
	}

	args := strings.TrimSpace(buffer.String())

	cmdOut, err := exec.Command("./ow-bench.sh", args).Output()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(cmdOut))
}
