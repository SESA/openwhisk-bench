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
var cmdChan = make(chan map[string]string)
var wgTime = sync.WaitGroup{}
var outputFileWriter os.File
var verbose = true
var debug = false

func main() {
	writeToFile := flag.Bool("writeToFile", false, "Write output to file")
	outputFilePath := flag.String("fileName", generateOutputFileName(), "Write output to file")
	isCreateFlag := flag.Bool("create", false, "Create functions before execution")
	isVerbose := flag.Bool("v", true, "Verbose output")
	isDebug := flag.Bool("debug", false, "Debug output")
	isQuiet := flag.Bool("q", false, "Quiet output (data only)")

	flag.Parse()

	if !*writeToFile {
		*outputFilePath = ""
	}
	if *isVerbose {
		verbose = true
	}
	if *isDebug {
		debug = true
		verbose = true
	}
	if *isQuiet {
		verbose = false
		debug = false
	}

	argsArr := flag.Args()

	if verbose {
		fmt.Println("WriteToFile: ", *writeToFile, ", FileName: ", *outputFilePath, ", Create: ", *isCreateFlag)
		fmt.Println("Command: " + argsArr[0])
	}

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
	os.Exit(0)
}

func execCmdsFromFile(inputFilePath string, outputFilePath string, needCreation bool) {
	if verbose {
		fmt.Println("Parsing File: " + inputFilePath)
	}
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

		if verbose {
			fmt.Println("Creation Needed: " + strconv.FormatBool(needCreation))
		}
		if needCreation {
			for user := range uniqueUsersList {
				userCreationResult := execCmd([]string{"createUser", user})
				userAuth := strings.Split(userCreationResult, " ")[1]
				userVsAuthMap[user] = userAuth
			}
			if verbose {
				fmt.Println("User Creation Done.")
			}
			for user, funcList := range usersVsFuncsMap {
				//userAuth := userVsAuthMap[user]
				for funcName := range funcList {
					execCmd([]string{"createFunction", user, strconv.Itoa(funcName), "funcs/iter.js"})
				}
			}
			if verbose {
				fmt.Println("Function Creation Done.")
			}
		} else {
			for user := range uniqueUsersList {
				userAuth := execCmd([]string{"getUserAuth", user})
				userVsAuthMap[user] = userAuth
			}
			if verbose {
				fmt.Println("User-Auth Map Loaded.")
			}
		}
	}

	if verbose {
		fmt.Println("Starting function invocations across " + strconv.Itoa(OPEN_WHISK_CONCURRENCY_FACTOR) + " co-routines:")
		fmt.Println("------------------------------------------------------------------------")
	}

	if outputFilePath != "" {
		outputFileWriter = createOutputFile(outputFilePath)
		outputFileWriter.WriteString(BATCH + ", ")
		outputFileWriter.WriteString(USER_ID + ", ")
		outputFileWriter.WriteString(FUNCTION_ID + ", ")
		outputFileWriter.WriteString(SEQ + ", ")
		outputFileWriter.WriteString(CMD_RESULT + ", ")
		outputFileWriter.WriteString(ELAPSED_TIME + ", ")
		outputFileWriter.WriteString(SUBMITTED_AT + ", ")
		outputFileWriter.WriteString(ENDED_AT + ", ")
		outputFileWriter.WriteString(PARAMETER + "\n")
	} else {
		fmt.Print(BATCH + ", ")
		fmt.Print(USER_ID + ", ")
		fmt.Print(FUNCTION_ID + ", ")
		fmt.Print(SEQ + ", ")
		fmt.Print(CMD_RESULT + ", ")
		fmt.Print(ELAPSED_TIME + ", ")
		fmt.Print(SUBMITTED_AT + ", ")
		fmt.Print(ENDED_AT + ", ")
		fmt.Print(PARAMETER + "\n")
	}

	batchArr := make([]int, 0, len(batchVsUserFuncMap))
	for batchOfExecution := range batchVsUserFuncMap {
		batchArr = append(batchArr, batchOfExecution)
	}
	sort.Ints(batchArr)

	for i := 0; i < OPEN_WHISK_CONCURRENCY_FACTOR; i++ {
		go invokeFunction(outputFilePath != "")
	}

	totalExecCount := 0
	startRun := time.Now()
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

		// Wait for all executions to finish
		wgTime.Wait()

		if verbose {
			batch_elapse := time.Since(startBatch)
			fmt.Println("------------------------------------------------------------------------")
			fmt.Println("Batch #"+strconv.Itoa(batchOfExecution)+" completed "+strconv.Itoa(batchExecCount)+" executions in", int(batch_elapse.Seconds()*1000), "ms")
			fmt.Println("------------------------------------------------------------------------")
		}
	}
	elapsed := time.Since(startRun)

	if verbose {
		fmt.Println("Total time:", int(elapsed.Seconds()*1000), "ms")
		fmt.Println("Total executions: " + strconv.Itoa(totalExecCount))
	}
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
	if debug {
		fmt.Println(args)
	}
	cmdOut, err := exec.Command("./ow-bench.sh", args).Output()
	if err != nil {
		log.Fatal(err)
	}

	return strings.Trim(string(cmdOut), " \n")
}

func invokeFunction(writeToFile bool) {
	for cmdMap := range cmdChan {
		userAuth := cmdMap[USER_AUTH]
		functionID := cmdMap[FUNCTION_ID]
		param := cmdMap[PARAMETER]

		start := time.Now().UnixNano()
		paramArr := []string{"invokeFunctionWithAuth", userAuth, functionID}
		if param != "" {
			paramArr = append(paramArr, "--param", param)
		}
		execResult := execCmd(paramArr)
		end := time.Now().UnixNano()
		elapsed := ((end - start) / 1000000) /* nano to milli */

		resultMap := copyMap(cmdMap)
		delete(resultMap, USER_AUTH)
		resultMap[CMD_RESULT] = execResult
		resultMap[SUBMITTED_AT] = strconv.FormatInt(start, 10)
		resultMap[ENDED_AT] = strconv.FormatInt(end, 10)
		resultMap[ELAPSED_TIME] = strconv.FormatInt(elapsed, 10)

		orderArr := []string{BATCH, USER_ID, FUNCTION_ID, SEQ, CMD_RESULT, ELAPSED_TIME, SUBMITTED_AT, ENDED_AT, PARAMETER}
		if writeToFile {
			writeMapToFile(outputFileWriter, resultMap, orderArr)
		} else {
			writeMapToOut(resultMap, orderArr)
		}

		wgTime.Done()

		if strings.HasPrefix(execResult, "error") {
			panic(fmt.Errorf("Error during execution - %s", execResult))
		}
	}
}
