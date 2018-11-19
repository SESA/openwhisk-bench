package openwhisk

import (
	"../commons"
	"bufio"
	"bytes"
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

var currExecRate float64
var startRun time.Time
var IsAsync = false
var execCount = 0

var orderArr = []string{commons.BATCH, commons.USER_ID, commons.FUNCTION_ID, commons.SEQ, commons.CMD_RESULT, commons.ELAPSED_TIME, commons.ELAPSED_TIME_SINCE_START, commons.SUBMITTED_AT, commons.ENDED_AT, commons.EXEC_RATE, commons.CMD_STATUS, commons.CONCURRENCY_FACTOR, commons.PARAMETER}

func ExecCmdsFromFile(inputFilePath string, outputFilePath string, needCreation bool) {
	commons.PrintToStdOutOnVerbose("Parsing File: " + inputFilePath)

	fread, err := os.Open(inputFilePath)
	if err != nil {
		panic(fmt.Errorf("File error - %s", err))
	}

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

		doInitialization(needCreation, uniqueUsersList, usersVsFuncsMap)
	}

	commons.PrintToStdOutOnVerbose("Starting function invocations across " + strconv.Itoa(commons.ConcurrencyFactor) + " co-routines:")
	commons.PrintToStdOutOnVerbose("------------------------------------------------------------------------")

	if outputFilePath != "" {
		outputFilePath = "openwhisk/" + outputFilePath
		commons.OutputFileWriter = commons.CreateOutputFile(outputFilePath)
	}

	commons.PrintHeader(orderArr, outputFilePath)

	batchArr := make([]int, 0, len(batchVsUserFuncMap))
	for batchOfExecution := range batchVsUserFuncMap {
		batchArr = append(batchArr, batchOfExecution)
	}
	sort.Ints(batchArr)

	for i := 0; i < commons.ConcurrencyFactor; i++ {
		go invokeFunction()
	}

	if IsAsync {
		go getResult()
	}

	totalExecCount := 0
	startRun = time.Now()
	for {
		for _, batchOfExecution := range batchArr {
			batchExecCount := 0
			startBatch := time.Now()
			for _, userFuncObj := range batchVsUserFuncMap[batchOfExecution] {
				userAuth := userVsAuthMap[userFuncObj.UserID]
				for i := 1; i <= userFuncObj.NoOfTimesToExecute; i++ {
					cmdMap := make(map[string]string)
					cmdMap[commons.BATCH] = strconv.Itoa(batchOfExecution)
					cmdMap[commons.USER_ID] = userFuncObj.UserID
					cmdMap[commons.USER_AUTH] = userAuth
					cmdMap[commons.FUNCTION_ID] = strconv.Itoa(userFuncObj.FunctionID)
					cmdMap[commons.PARAMETER] = userFuncObj.Param
					cmdMap[commons.SEQ] = strconv.Itoa(totalExecCount)
					wgTime.Add(1)
					batchExecCount++
					totalExecCount++

					if commons.RateLimit != 0.0 && currExecRate > commons.RateLimit {
						//sleepTime := int((currExecRate/(rateLimit*5))*1000)
						//fmt.Println("Exec Count: " + strconv.Itoa(execCount) + ", Seq: " + strconv.Itoa(totalExecCount) + ", Exec Rate: " + strconv.FormatFloat(currExecRate, 'f', 2, 64) + ", Sleep Time: " + strconv.Itoa(sleepTime))
						//time.Sleep(time.Duration(sleepTime) * time.Millisecond)
						time.Sleep(500 * time.Millisecond)
					}

					cmdChan <- cmdMap
				}
			}

			wgTime.Wait()

			batchElapse := time.Since(startBatch)
			commons.PrintToStdOutOnVerbose("------------------------------------------------------------------------")
			commons.PrintToStdOutOnVerbose("Batch #" + strconv.Itoa(batchOfExecution) + " completed " + strconv.Itoa(batchExecCount) + " executions in " + strconv.FormatFloat(batchElapse.Seconds()*1000, 'f', 0, 64) + "  ms")
			commons.PrintToStdOutOnVerbose("------------------------------------------------------------------------")

		}

		if !commons.RunForever {
			break
		}
	}

	elapsed := time.Since(startRun)
	elapsedTimeInMs := elapsed.Seconds() * 1000

	commons.PrintToStdOutOnVerbose("Total time: " + strconv.FormatFloat(elapsedTimeInMs, 'f', 0, 64) + " ms")
	commons.PrintToStdOutOnVerbose("Total executions: " + strconv.Itoa(totalExecCount))
	commons.PrintToStdOutOnVerbose("Execution Rate: " + strconv.FormatFloat(float64(totalExecCount)/(elapsedTimeInMs/1000), 'f', 2, 64))

	commons.OutputFileWriter.Close()
}

/* execute single openwhisk cli command with argsArr arguments */
func ExecCmd(argsArr []string) string {
	var buffer bytes.Buffer

	for i := 0; i < len(argsArr); i++ {
		buffer.WriteString(argsArr[i])
		buffer.WriteString(" ")
	}

	args := strings.TrimSpace(buffer.String())
	commons.PrintToStdOutOnDebug(args)

	cmdOut, err := exec.Command("./openwhisk/ow-bench.sh", args).Output()
	if err != nil {
		log.Fatal(err)
	}

	return strings.Trim(string(cmdOut), " \n")
}

func processResult(resultMap map[string]string) {
	delete(resultMap, commons.USER_AUTH)
	elapsedTimeSinceStart := time.Since(startRun).Seconds() * 1000
	resultMap[commons.ELAPSED_TIME_SINCE_START] = strconv.FormatFloat(elapsedTimeSinceStart, 'f', 0, 64)
	resultMap[commons.CONCURRENCY_FACTOR] = strconv.Itoa(commons.ConcurrencyFactor)

	counterMtx.Lock()
	execCount += 1
	currExecRate = float64(execCount) / (elapsedTimeSinceStart / 1000)
	resultMap[commons.EXEC_RATE] = strconv.FormatFloat(currExecRate, 'f', 2, 64)
	counterMtx.Unlock()

	if commons.WriteToFile {
		commons.WriteMapToFile(resultMap, orderArr)
	} else {
		commons.WriteMapToOut(resultMap, orderArr)
	}
}

func doExecAndParse(paramArr []string, retryCount int) string {
	jsonStr := ExecCmd(paramArr)
	_, parsedJson := commons.ParseJsonResponse(jsonStr)
	if strings.Contains(parsedJson, "request timed out") {
		if retryCount > 0 {
			doExecAndParse(paramArr, retryCount-1)
		} else {
			panic(fmt.Errorf("Timeout error - %s", parsedJson))
		}
	}

	return parsedJson
}

func doInitialization(needCreation bool, uniqueUsersList map[string]struct{}, usersVsFuncsMap map[string]map[int]struct{}) {
	commons.PrintToStdOutOnVerbose("Creation Needed: " + strconv.FormatBool(needCreation))

	var concChan = make(chan int, commons.ConcurrencyFactor)

	if needCreation {
		startTime := time.Now()
		for user := range uniqueUsersList {
			concChan <- 1
			wgTime.Add(1)

			go func(user string) {
				parsedJson := doExecAndParse([]string{"createUser", user}, 10)
				userAuth := strings.Split(parsedJson, " ")[1]

				counterMtx.Lock()
				userVsAuthMap[user] = userAuth
				counterMtx.Unlock()

				wgTime.Done()
				<-concChan
			}(user)
		}

		wgTime.Wait()
		commons.PrintToStdOutOnVerbose(strconv.Itoa(len(uniqueUsersList)) + " users created. Time taken = " + time.Since(startTime).String())

		totalFuncsCreated := 0
		startTime = time.Now()
		for user, funcList := range usersVsFuncsMap {
			//userAuth := userVsAuthMap[user]
			for funcName := range funcList {
				concChan <- 1
				wgTime.Add(1)

				go func(user string, funcName int) {
					doExecAndParse([]string{"createFunction", user, strconv.Itoa(funcName), "openwhisk/funcs/spin.js"}, 5)

					wgTime.Done()
					<-concChan
				}(user, funcName)
			}

			totalFuncsCreated += len(funcList)
			commons.PrintToStdOutOnDebug(strconv.Itoa(len(funcList)) + " functions created for " + user)
		}

		wgTime.Wait()
		commons.PrintToStdOutOnVerbose(strconv.Itoa(totalFuncsCreated) + " functions created. Time taken = " + time.Since(startTime).String())
	} else {
		startTime := time.Now()
		for user := range uniqueUsersList {
			concChan <- 1
			wgTime.Add(1)

			go func(user string) {
				userAuth := doExecAndParse([]string{"getUserAuth", user}, 10)

				counterMtx.Lock()
				userVsAuthMap[user] = userAuth
				counterMtx.Unlock()

				wgTime.Done()
				<-concChan
			}(user)
		}

		wgTime.Wait()
		commons.PrintToStdOutOnVerbose(strconv.Itoa(len(uniqueUsersList)) + " users are loaded with their auth details. Time taken = " + time.Since(startTime).String())
	}
}

func invokeFunction() {
	for cmdMap := range cmdChan {
		userAuth := cmdMap[commons.USER_AUTH]
		functionID := cmdMap[commons.FUNCTION_ID]
		param := cmdMap[commons.PARAMETER]

		start := time.Now().UnixNano()

		cmd := "invokeFunctionWithAuth"
		if IsAsync {
			cmd = "invokeFunctionWithAuthAsync"
		}

		var paramArr []string
		if IsAsync {
			paramArr = []string{cmd, userAuth, functionID}
		} else {
			paramArr = []string{cmd, "false", userAuth, functionID}
		}

		if param != "" {
			paramArr = append(paramArr, "--param", param)
		}

		jsonStr := ExecCmd(paramArr)
		status, execResult := commons.ParseJsonResponse(jsonStr)

		end := time.Now().UnixNano()
		elapsed := (end - start) / 1000000 /* nano to milli */

		resultMap := commons.CopyMap(cmdMap)
		resultMap[commons.CMD_STATUS] = status
		resultMap[commons.CMD_RESULT] = execResult
		resultMap[commons.SUBMITTED_AT] = strconv.FormatInt(start, 10)

		if IsAsync {
			activationList = append(activationList, resultMap)
		} else {
			resultMap[commons.ENDED_AT] = strconv.FormatInt(end, 10)
			resultMap[commons.ELAPSED_TIME] = strconv.FormatInt(elapsed, 10)
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
			userAuth := resultMap[commons.USER_AUTH]
			activationID := resultMap[commons.CMD_RESULT]

			paramArr := []string{"getResultFromActivation", userAuth, activationID}
			jsonStr := ExecCmd(paramArr)
			status, execResult := commons.ParseJsonResponse(jsonStr)

			if execResult != "-1, -1, -1, -1" && len(strings.Split(execResult, ", ")) == 4 {
				start, _ := strconv.ParseInt(resultMap[commons.ELAPSED_TIME], 10, 64)
				end := time.Now().UnixNano()
				elapsed := (end - start) / 1000000 /* nano to milli */

				resultMap[commons.CMD_STATUS] = status
				resultMap[commons.CMD_RESULT] = execResult
				resultMap[commons.ENDED_AT] = strconv.FormatInt(end, 10)
				resultMap[commons.ELAPSED_TIME] = strconv.FormatInt(elapsed, 10)
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
