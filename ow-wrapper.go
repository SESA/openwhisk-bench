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
)

var userVsAuthMap = make(map[string]string)

func main() {
	argsArr := make([]string, len(os.Args)-2)
	copy(argsArr, os.Args[2:])

	if os.Args[1] == "execCmd" {
		fmt.Println(execCmd(argsArr))
	} else if os.Args[1] == "execFile" {
		execCmdsFromFile(argsArr[0], len(argsArr) == 2 && argsArr[1] == "-create")
	}
}

func execCmdsFromFile(filePath string, needCreation bool) {
	fmt.Println(filePath)
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

		if needCreation {
			for user := range uniqueUsersList {
				userCreationResult := execCmd([]string{"createUser", user})
				userAuth := strings.Split(userCreationResult, " ")[1]
				userVsAuthMap[user] = userAuth
			}

			for user, funcList := range usersVsFuncsMap {
				userAuth := userVsAuthMap[user]
				for funcName := range funcList {
					execCmd([]string{"createFunction", userAuth, strconv.Itoa(funcName)})
				}
			}
		} else {
			for user := range uniqueUsersList {
				userAuth := execCmd([]string{"getUserAuth", user})
				userVsAuthMap[user] = userAuth
			}
		}
	}

	fmt.Println("Started Invoking Functions")
	timeArr := make([]int, 0, len(timeVsUserFuncMap))
	for time := range timeVsUserFuncMap {
		timeArr = append(timeArr, time)
	}

	sort.Ints(timeArr)

	for _, time := range timeArr {
		for _, userFuncObj := range timeVsUserFuncMap[time] {
			go invokeFunction(userFuncObj)
		}
	}
}

func execCmd(argsArr []string) string {
	var buffer bytes.Buffer

	for i := 0; i < len(argsArr); i++ {
		buffer.WriteString(argsArr[i])
		buffer.WriteString(" ")
	}

	args := strings.TrimSpace(buffer.String())
	cmdOut, err := exec.Command("./ow-bench.sh", args).Output()
	if err != nil {
		fmt.Println(err)
	}

	return strings.Trim(string(cmdOut), " \n")
}

func invokeFunction(userFuncObj UserFuncs) {
	userAuth := userVsAuthMap[userFuncObj.UserID]
	for i := 0; i < userFuncObj.NoOfTimesToExecute; i++ {
		execResult := execCmd([]string{"invokeFunction", strconv.Itoa(userFuncObj.FunctionID), userAuth})
		fmt.Println(execResult)
	}
}
