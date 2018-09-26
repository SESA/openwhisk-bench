package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var userVsAuthMap = make(map[string]string)
var activationList []map[string]string
var cmdChan = make(chan map[string]string)

var wgTime = sync.WaitGroup{}
var counterMtx sync.Mutex
var outputFileWriter os.File

var writeToFile bool
var debug bool
var startRun time.Time
var verbose = true
var isAsync = false
var execCount = 0

var orderArr = []string{BATCH, USER_ID, FUNCTION_ID, SEQ, CMD_RESULT, ELAPSED_TIME, ELAPSED_TIME_SINCE_START, SUBMITTED_AT, ENDED_AT, EXEC_RATE, PARAMETER}

func main() {
	outputFilePath := flag.String("fileName", generateOutputFileName(), "Write output to file")
	isCreateFlag := flag.Bool("create", false, "Create functions before execution")
	isQuiet := flag.Bool("q", false, "Quiet output (data only)")

	flag.BoolVar(&writeToFile, "writeToFile", false, "Write output to file")
	flag.BoolVar(&verbose, "v", true, "Verbose output")
	flag.BoolVar(&debug, "debug", false, "Debug output")
	flag.BoolVar(&isAsync, "async", false, "Invoke functions asynchronously")

	flag.Parse()

	if !writeToFile {
		*outputFilePath = ""
	}

	if debug {
		verbose = true
	}

	if *isQuiet {
		verbose = false
		debug = false
	}

	argsArr := flag.Args()

	printToStdOutOnVerbose("WriteToFile: " + strconv.FormatBool(writeToFile) + ", FileName: " + *outputFilePath + ", Create: " + strconv.FormatBool(*isCreateFlag) + ", Verbose: " + strconv.FormatBool(verbose) + ", Debug: " + strconv.FormatBool(debug) + ", Quiet: " + strconv.FormatBool(*isQuiet) + ", Async: " + strconv.FormatBool(isAsync))
	printToStdOutOnVerbose("Command: " + argsArr[0])

	/* Main Benchmark Methods */
	switch argsArr[0] {
	case "execCmd":
		fmt.Println(execCmd(argsArr[1:]))
	case "execFile":
		execCmdsFromFile(argsArr[1], *outputFilePath, *isCreateFlag)
	default:
		fmt.Println("Command not found: " + argsArr[0])
		fmt.Println("Exiting")
		os.Exit(127)
	}

	printToStdOutOnVerbose("Execution Complete")
	os.Exit(0)
}

func execCmdsFromFile(inputFilePath string, outputFilePath string, needCreation bool) {
	printToStdOutOnVerbose("Parsing File: " + inputFilePath)

	fread, _ := os.Open(inputFilePath)
	scanner := bufio.NewScanner(fread)

	batchVsUserFuncMap := make(map[int][]UserFuncs)

	{
		var exists = struct{}{}
		uniqueUsersList := make(map[string]struct{})
		usersVsFuncsMap := make(map[string]map[int]struct{})

		for scanner.Scan() {
			lineParts := strings.Split(scanner.Text(), ",")
			userFuncObj := createUserFuncsObj(lineParts)
			userFuncArr := batchVsUserFuncMap[userFuncObj.Time]
			userFuncArr = append(userFuncArr, userFuncObj)
			batchVsUserFuncMap[userFuncObj.Time] = userFuncArr
			uniqueUsersList[userFuncObj.UserID] = exists

			if needCreation {
				uniqueFuncList, ok := usersVsFuncsMap[userFuncObj.UserID]
				if !ok {
					uniqueFuncList = make(map[int]struct{})
				}

				uniqueFuncList[userFuncObj.FunctionID] = exists
				usersVsFuncsMap[userFuncObj.UserID] = uniqueFuncList
			}
		}

		printToStdOutOnVerbose("Creation Needed: " + strconv.FormatBool(needCreation))

		if needCreation {
			for user := range uniqueUsersList {
				userCreationResult := execCmd([]string{"createUser", user})
				userAuth := strings.Split(userCreationResult, " ")[1]
				userVsAuthMap[user] = userAuth
			}

			printToStdOutOnVerbose("User Creation Done.")

			for user, funcList := range usersVsFuncsMap {
				//userAuth := userVsAuthMap[user]
				for funcName := range funcList {
					execCmd([]string{"createFunction", user, strconv.Itoa(funcName), "funcs/iter.js"})
				}
			}

			printToStdOutOnVerbose("Function Creation Done.")

		} else {
			for user := range uniqueUsersList {
				userAuth := execCmd([]string{"getUserAuth", user})
				userVsAuthMap[user] = userAuth
			}

			printToStdOutOnVerbose("User-Auth Map Loaded.")

		}
	}

	printToStdOutOnVerbose("Starting function invocations across " + strconv.Itoa(OPEN_WHISK_CONCURRENCY_FACTOR) + " co-routines:")
	printToStdOutOnVerbose("------------------------------------------------------------------------")

	if outputFilePath != "" {
		outputFileWriter = createOutputFile(outputFilePath)
	}

	for i := 0; i < len(orderArr); i++ {
		delimiter := ", "
		if i == len(orderArr)-1 {
			delimiter = "\n"
		}

		if outputFilePath != "" {
			outputFileWriter.WriteString(orderArr[i] + delimiter)
		} else {
			fmt.Print(orderArr[i] + delimiter)
		}
	}

	batchArr := make([]int, 0, len(batchVsUserFuncMap))
	for batchOfExecution := range batchVsUserFuncMap {
		batchArr = append(batchArr, batchOfExecution)
	}
	sort.Ints(batchArr)

	for i := 0; i < OPEN_WHISK_CONCURRENCY_FACTOR; i++ {
		go invokeFunction()
	}

	if isAsync {
		go getResult()
	}

	totalExecCount := 0
	startRun = time.Now()
	for _, batchOfExecution := range batchArr {
		batchExecCount := 0
		startBatch := time.Now()
		for _, userFuncObj := range batchVsUserFuncMap[batchOfExecution] {
			userAuth := userVsAuthMap[userFuncObj.UserID]
			for i := 1; i <= userFuncObj.NoOfTimesToExecute; i++ {
				cmdMap := make(map[string]string)
				cmdMap[BATCH] = strconv.Itoa(batchOfExecution)
				cmdMap[USER_ID] = userFuncObj.UserID
				cmdMap[USER_AUTH] = userAuth
				cmdMap[FUNCTION_ID] = strconv.Itoa(userFuncObj.FunctionID)
				cmdMap[PARAMETER] = userFuncObj.Param
				cmdMap[SEQ] = strconv.Itoa(totalExecCount)
				wgTime.Add(1)
				batchExecCount++
				totalExecCount++
				cmdChan <- cmdMap
			}
		}

		wgTime.Wait()

		batchElapse := time.Since(startBatch)
		printToStdOutOnVerbose("------------------------------------------------------------------------")
		printToStdOutOnVerbose("Batch #" + strconv.Itoa(batchOfExecution) + " completed " + strconv.Itoa(batchExecCount) + " executions in" + strconv.FormatFloat(batchElapse.Seconds()*1000, 'f', 0, 64) + "  ms")
		printToStdOutOnVerbose("------------------------------------------------------------------------")

	}

	elapsed := time.Since(startRun)
	elapsedTimeInMs := elapsed.Seconds() * 1000

	printToStdOutOnVerbose("Total time: " + strconv.FormatFloat(elapsedTimeInMs, 'f', 0, 64) + " ms")
	printToStdOutOnVerbose("Total executions: " + strconv.Itoa(totalExecCount))
	printToStdOutOnVerbose("Execution Rate: " + strconv.FormatFloat(float64(totalExecCount)/(elapsedTimeInMs/1000), 'f', 2, 64))

	outputFileWriter.Close()
}

/* execute single openwhisk cli command with argsArr arguments */
func execCmd(argsArr []string) string {
	var buffer bytes.Buffer

	for i := 0; i < len(argsArr); i++ {
		buffer.WriteString(argsArr[i])
		buffer.WriteString(" ")
	}

	args := strings.TrimSpace(buffer.String())
	printToStdOutOnDebug(args)

	cmdOut, err := exec.Command("./ow-bench.sh", args).Output()
	if err != nil {
		log.Fatal(err)
	}

	return strings.Trim(string(cmdOut), " \n")
}

func processResult(resultMap map[string]string) {
	delete(resultMap, USER_AUTH)
	elapsedTimeSinceStart := time.Since(startRun).Seconds() * 1000
	resultMap[ELAPSED_TIME_SINCE_START] = strconv.FormatFloat(elapsedTimeSinceStart, 'f', 0, 64)

	counterMtx.Lock()
	execCount += 1
	resultMap[EXEC_RATE] = strconv.FormatFloat(float64(execCount)/(elapsedTimeSinceStart/1000), 'f', 2, 64)
	counterMtx.Unlock()

	if writeToFile {
		writeMapToFile(outputFileWriter, resultMap, orderArr)
	} else {
		writeMapToOut(resultMap, orderArr)
	}
}

func invokeFunction() {
	for cmdMap := range cmdChan {
		userAuth := cmdMap[USER_AUTH]
		functionID := cmdMap[FUNCTION_ID]
		param := cmdMap[PARAMETER]

		start := time.Now().UnixNano()

		cmd := "invokeFunctionWithAuth"
		if isAsync {
			cmd = "invokeFunctionWithAuthAsync"
		}

		var paramArr []string
		if isAsync {
			paramArr = []string{cmd, userAuth, functionID}
		} else {
			paramArr = []string{cmd, "false", userAuth, functionID}
		}

		if param != "" {
			paramArr = append(paramArr, "--param", param)
		}

		execResult := execCmd(paramArr)
		end := time.Now().UnixNano()
		elapsed := (end - start) / 1000000 /* nano to milli */

		resultMap := copyMap(cmdMap)
		resultMap[CMD_RESULT] = execResult
		resultMap[SUBMITTED_AT] = strconv.FormatInt(start, 10)

		if isAsync {
			activationList = append(activationList, resultMap)
		} else {
			resultMap[ENDED_AT] = strconv.FormatInt(end, 10)
			resultMap[ELAPSED_TIME] = strconv.FormatInt(elapsed, 10)
			processResult(resultMap)
			wgTime.Done()
		}

		if strings.HasPrefix(execResult, "error") {
			panic(fmt.Errorf("Error during execution - %s", execResult))
		}
	}
}

func getResult() {
	for {
		for idx := len(activationList) - 1; idx >= 0; idx-- {
			resultMap := activationList[idx]
			userAuth := resultMap[USER_AUTH]
			activationID := resultMap[CMD_RESULT]

			paramArr := []string{"getResultFromActivation", userAuth, activationID}
			execResult := execCmd(paramArr)

			if execResult != "-1, -1, -1, -1" && len(strings.Split(execResult, ", ")) == 4 {
				start, _ := strconv.ParseInt(resultMap[ELAPSED_TIME], 10, 64)
				end := time.Now().UnixNano()
				elapsed := (end - start) / 1000000 /* nano to milli */

				resultMap[CMD_RESULT] = execResult
				resultMap[ENDED_AT] = strconv.FormatInt(end, 10)
				resultMap[ELAPSED_TIME] = strconv.FormatInt(elapsed, 10)
				processResult(resultMap)

				lastIdx := len(activationList) - 1
				activationList[idx] = activationList[lastIdx]
				activationList = activationList[:lastIdx]
				wgTime.Done()
			}
		}

		time.Sleep(2 * time.Second)
	}
}
