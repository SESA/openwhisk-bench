package main

import (
	"fmt"
	"os"
	"bufio"
	"os/exec"
	"bytes"
	"strings"
	"strconv"
	"sort"
	"sync"
	"log"
	"time"
		)

var userVsAuthMap = make(map[string]string)
var cmdChan = make(chan map[string]string)
var wgTime = sync.WaitGroup{}
var outputFileWriter os.File

func main() {
	/* initialize array of command line arguments */
	argsArr := make([]string, len(os.Args)-2)
	copy(argsArr, os.Args[2:])

	fmt.Println("Received Command: " + os.Args[1])

	/* execute single openwhisk cli command */
	if os.Args[1] == "execCmd" {
		fmt.Println(execCmd(argsArr))
	/* execute multiple openwhisk cli commands from file */
	} else if os.Args[1] == "execFile" {
		execCmdsFromFile(argsArr[0], len(argsArr) == 2 && argsArr[1] == "-create")
	}

	fmt.Println("Execution Completed.")
}

func execCmdsFromFile(filePath string, needCreation bool) {
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

			if needCreation {
				uniqueUsersList[userFuncObj.UserID] = exists
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
				userAuth := userVsAuthMap[user]
				for funcName := range funcList {
					execCmd([]string{"createFunction", userAuth, strconv.Itoa(funcName)})
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

	outputFileWriter = createOutputFile(filePath)
	outputFileWriter.WriteString(TIME + ", ")
	outputFileWriter.WriteString(USER_ID + ", ")
	outputFileWriter.WriteString(FUNCTION_ID + ", ")
	outputFileWriter.WriteString(SEQ + ", ")
	outputFileWriter.WriteString(CMD_RESULT + ", ")
	outputFileWriter.WriteString(SUBMITTED_AT + ", ")
	outputFileWriter.WriteString(ENDED_AT + ", ")
	outputFileWriter.WriteString(ELAPSED_TIME_IN_NS + ", ")
	outputFileWriter.WriteString(ELAPSED_TIME_IN_SEC + "\n")

	fmt.Println("Started Invoking Functions")
	timeArr := make([]int, 0, len(timeVsUserFuncMap))
	for timeOfExecution := range timeVsUserFuncMap {
		timeArr = append(timeArr, timeOfExecution)
	}

	sort.Ints(timeArr)

	fmt.Println("Spanning " + strconv.Itoa(OPEN_WHISK_CONCURRENCY_FACTOR) + " co-routines to handle jobs")
	for i := 0; i < OPEN_WHISK_CONCURRENCY_FACTOR; i++ {
		go invokeFunction()
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
				cmdMap[SEQ] = strconv.Itoa(i)

				wgTime.Add(1)
				cmdChan <- cmdMap
			}
		}

		wgTime.Wait()
		fmt.Println("Time " + strconv.Itoa(timeOfExecution) + " jobs completed.")
	}
	elapsed := time.Since(start)

	fmt.Println("Total Job Time: ", elapsed.String())

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

	cmdOut, err := exec.Command("./ow-bench.sh", args).Output()
	if err != nil {
		log.Fatal(err)
	}

	return strings.Trim(string(cmdOut), " \n")
}

func invokeFunction() {
	for cmdMap := range cmdChan {
		//userAuth := cmdMap[USER_AUTH]
		functionID := cmdMap[FUNCTION_ID]
                userAuth := "guest"

		start := time.Now()
		execResult := execCmd([]string{"invokeFunction", userAuth, functionID})
		elapsed := time.Since(start)

		resultMap := copyMap(cmdMap)
		delete(resultMap, USER_AUTH)
		resultMap[CMD_RESULT] = execResult
		resultMap[SUBMITTED_AT] = start.String()
		resultMap[ENDED_AT] = start.Add(elapsed).String()
		resultMap[ELAPSED_TIME_IN_NS] = strconv.FormatInt(elapsed.Nanoseconds(), 10)
		resultMap[ELAPSED_TIME_IN_SEC] = strconv.Itoa(int(elapsed.Seconds()))

		writeMapToFile(outputFileWriter, resultMap, []string{TIME, USER_ID, FUNCTION_ID, SEQ, CMD_RESULT, SUBMITTED_AT, ENDED_AT, ELAPSED_TIME_IN_NS, ELAPSED_TIME_IN_SEC})
		wgTime.Done()
	}
}