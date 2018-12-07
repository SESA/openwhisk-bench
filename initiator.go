package main

import (
	"./commons"
	"./docker"
	"./openwhisk"
	"flag"
	"fmt"
	"os"
	"strconv"
)

func main() {
	// Common flags for open-whisk & docker
	outputFilePath := flag.String("fileName", commons.GenerateOutputFileName(), "Write output to file")
	isQuiet := flag.Bool("q", false, "Quiet output (data only)")

	flag.BoolVar(&commons.WriteToFile, "writeToFile", false, "Write output to file")
	flag.BoolVar(&commons.Verbose, "v", true, "Verbose output")
	flag.BoolVar(&commons.Debug, "debug", false, "Debug output")
	flag.BoolVar(&commons.RunForever, "forever", false, "Run forever till the user sends stop signal")
	flag.IntVar(&commons.ConcurrencyFactor, "cf", commons.OPEN_WHISK_CONCURRENCY_FACTOR, "Sets OpenWhisk Concurrency Factor (Creates N co-routines to spawn commands to OpenWhisk")

	// Flags for open-whisk
	flag.Float64Var(&commons.RateLimit, "rateLimit", 0, "Rate Limiter to maintain the execution rate")
	isCreateFlag := flag.Bool("create", false, "Create functions before execution")
	flag.BoolVar(&openwhisk.IsAsync, "async", false, "Invoke functions asynchronously")

	// Flags for docker
	flag.IntVar(&docker.CheckMemStats, "memCheckInterval", -1, "Check Memory Stats Periodically")

	flag.Parse()

	if !commons.WriteToFile {
		*outputFilePath = ""
	}

	if commons.Debug {
		commons.Verbose = true
	}

	if *isQuiet {
		commons.Verbose = false
		commons.Debug = false
	}

	argsArr := flag.Args()

	commons.PrintToStdOutOnVerbose("WriteToFile: " + strconv.FormatBool(commons.WriteToFile) + ", FileName: " + *outputFilePath + ", Create: " + strconv.FormatBool(*isCreateFlag) + ", Verbose: " + strconv.FormatBool(commons.Verbose) + ", Debug: " + strconv.FormatBool(commons.Debug) + ", Quiet: " + strconv.FormatBool(*isQuiet) + ", Async: " + strconv.FormatBool(openwhisk.IsAsync))
	commons.PrintToStdOutOnVerbose("Command: " + argsArr[0])

	/* Main Benchmark Methods */
	switch argsArr[0] {
	case "execOWCmd":
		fmt.Println(openwhisk.ExecCmd(argsArr[1:]))
	case "execOWFile":
		openwhisk.ExecCmdsFromFile(argsArr[1], *outputFilePath, *isCreateFlag)
	case "execDockerCmd":
		fmt.Println(docker.ExecCmd(argsArr[1:]))
	case "execDockerFile":
		docker.ExecCmdsFromFile(argsArr[1], *outputFilePath)
	case "testDockerCreateForever":
		docker.TestCreationForever(*outputFilePath, argsArr[1])
	default:
		fmt.Println("Command not found: " + argsArr[0])
		fmt.Println("Exiting")
		os.Exit(127)
	}

	commons.PrintToStdOutOnVerbose("Execution Complete")
	os.Exit(0)
}
