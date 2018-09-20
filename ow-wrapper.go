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

func main() {
	isWriteToFile := flag.Bool("writeToFile", false, "Write output to file")
	isCreateFlag := flag.Bool("create", false, "Create functions before execution")

	flag.Parse()
	fmt.Println("WriteToFile: ", *isWriteToFile, ", Create: ", *isCreateFlag)
	argsArr := flag.Args()

	fmt.Println("Received Command: " + argsArr[0])

	/* execute single openwhisk cli command */
	if argsArr[0] == "execCmd" {
		fmt.Println(execCmd(argsArr[1:]))
		/* execute multiple openwhisk cli commands from file */
	} else if argsArr[0] == "execFile" {
		execCmdsFromFile(argsArr[1], *isWriteToFile, *isCreateFlag)
	}

	fmt.Println("Execution Completed.")
}

func execCmdsFromFile(filePath string, writeToFile bool, needCreation bool) {
	fmt.Println("Parsing File: " + filePath)
	fread, _ := os.Open(filePath)
	scanner := bufio.NewScanner(fread)

	timeVsUserFuncMap := make(map[int][]UserFuncs)

	{
		var exists = struct{}{}
		uniqueUsersList := make(map[string]struct{})
		usersVsFuncsMap := make(map[string]map[int]struct{})

		for scanner.Scan() {
			lineParts := strings.Split(scanner.Text(), ",")
			userFuncObj := createUserFuncsObj(lineParts)
			userFuncArr := timeVsUserFuncMap[userFuncObj.Time]
			userFuncArr = append(userFuncArr, userFuncObj)
			timeVsUserFuncMap[userFuncObj.Time] = userFuncArr
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

		fmt.Println("Creation Needed: " + strconv.FormatBool(needCreation))
		if needCreation {
			for user := range uniqueUsersList {
				userCreationResult := execCmd([]string{"createUser", user})
				userAuth := strings.Split(userCreationResult, " ")[1]
				userVsAuthMap[user] = userAuth
			}
			fmt.Println("User Creation Done.")

			for user, funcList := range usersVsFuncsMap {
				//userAuth := userVsAuthMap[user]
				for funcName := range funcList {
					execCmd([]string{"createFunction", user, strconv.Itoa(funcName)})
				}
			}
			fmt.Println("Function Creation Done.")
		} else {
			for user := range uniqueUsersList {
				userAuth := execCmd([]string{"getUserAuth", user})
				userVsAuthMap[user] = userAuth
			}
			fmt.Println("User-Auth Map Loaded.")
		}
	}

	if writeToFile {
		outputFileWriter = createOutputFile(filePath)
		outputFileWriter.WriteString(TIME + ", ")
		outputFileWriter.WriteString(USER_ID + ", ")
		outputFileWriter.WriteString(FUNCTION_ID + ", ")
		outputFileWriter.WriteString(SEQ + ", ")
		outputFileWriter.WriteString(CMD_RESULT + ", ")
		outputFileWriter.WriteString(ELAPSED_TIME + ", ")
		outputFileWriter.WriteString(SUBMITTED_AT + ", ")
		outputFileWriter.WriteString(ENDED_AT + ", ")
		outputFileWriter.WriteString(PARAMETER + "\n")
	}

	fmt.Println("Started Invoking Functions")
	timeArr := make([]int, 0, len(timeVsUserFuncMap))
	for timeOfExecution := range timeVsUserFuncMap {
		timeArr = append(timeArr, timeOfExecution)
	}

	sort.Ints(timeArr)

	fmt.Println("Spanning " + strconv.Itoa(OPEN_WHISK_CONCURRENCY_FACTOR) + " co-routines to handle jobs")
	for i := 0; i < OPEN_WHISK_CONCURRENCY_FACTOR; i++ {
		go invokeFunction(writeToFile)
	}

	start := time.Now()
	for _, timeOfExecution := range timeArr {
		fmt.Println("Submitting jobs at time " + strconv.Itoa(timeOfExecution))

		for _, userFuncObj := range timeVsUserFuncMap[timeOfExecution] {
			userAuth := userVsAuthMap[userFuncObj.UserID]

			for i := 1; i <= userFuncObj.NoOfTimesToExecute; i++ {
				cmdMap := make(map[string]string)
				cmdMap[TIME] = strconv.Itoa(timeOfExecution)
				cmdMap[USER_ID] = userFuncObj.UserID
				cmdMap[USER_AUTH] = userAuth
				cmdMap[FUNCTION_ID] = strconv.Itoa(userFuncObj.FunctionID)
				cmdMap[PARAMETER] = userFuncObj.Param
				cmdMap[SEQ] = strconv.Itoa(i)

				wgTime.Add(1)
				cmdChan <- cmdMap
			}
		}

		wgTime.Wait()
		fmt.Println("Time " + strconv.Itoa(timeOfExecution) + " jobs completed.")
	}
	elapsed := time.Since(start)

	fmt.Println("Total Job Time:", int(elapsed.Seconds()*1000))

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
	//fmt.Println(args)
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

		orderArr := []string{TIME, USER_ID, FUNCTION_ID, SEQ, CMD_RESULT, ELAPSED_TIME, SUBMITTED_AT, ENDED_AT, PARAMETER}
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
