package docker

import (
	"../commons"
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var cmdChan = make(chan map[string]string)
var containerPrevCmdMap = make(map[string]string)

var wgTime = sync.WaitGroup{}
var counterMtx sync.Mutex

var currExecRate float64
var startRun time.Time
var execCount = 0

var orderArr = []string{commons.BATCH, commons.SEQ, commons.CONTAINER_NAME, commons.DOCKER_CMD, commons.ELAPSED_TIME, commons.ELAPSED_TIME_SINCE_START, commons.SUBMITTED_AT, commons.ENDED_AT, commons.EXEC_RATE, commons.CONCURRENCY_FACTOR, commons.PARAMETER}

func ExecCmdsFromFile(inputFilePath string, outputFilePath string) {
	defer cleanUpDocker()
	commons.PrintToStdOutOnVerbose("Parsing File: " + inputFilePath)

	fread, err := os.Open(inputFilePath)
	if err != nil {
		panic(fmt.Errorf("File error - %s", err))
	}

	scanner := bufio.NewScanner(fread)

	parseYAML()
	batchVsDockerFuncMap := make(map[int][]DockerFuncs)

	{
		for scanner.Scan() {
			lineParts := strings.Split(scanner.Text(), ",")
			dockerFuncObj := createDockerFuncsObj(lineParts)
			dockerFuncArr := batchVsDockerFuncMap[dockerFuncObj.Seq]
			dockerFuncArr = append(dockerFuncArr, dockerFuncObj)
			batchVsDockerFuncMap[dockerFuncObj.Seq] = dockerFuncArr
		}
	}

	commons.PrintToStdOutOnVerbose("Starting docker benchmark across " + strconv.Itoa(commons.ConcurrencyFactor) + " co-routines:")
	commons.PrintToStdOutOnVerbose("------------------------------------------------------------------------")

	if outputFilePath != "" {
		outputFilePath = "openwhisk/" + outputFilePath
		commons.OutputFileWriter = commons.CreateOutputFile(outputFilePath)
	}

	commons.PrintHeader(orderArr, outputFilePath)

	batchArr := make([]int, 0, len(batchVsDockerFuncMap))
	for batchOfExecution := range batchVsDockerFuncMap {
		batchArr = append(batchArr, batchOfExecution)
	}
	sort.Ints(batchArr)

	for i := 0; i < commons.ConcurrencyFactor; i++ {
		go invokeCommand()
	}

	totalExecCount := 0
	startRun = time.Now()
	for {
		for _, batchOfExecution := range batchArr {
			batchExecCount := 0
			startBatch := time.Now()
			for _, dockerFuncObj := range batchVsDockerFuncMap[batchOfExecution] {
				cmdMap := make(map[string]string)
				cmdMap[commons.BATCH] = strconv.Itoa(batchOfExecution)
				cmdMap[commons.CONTAINER_NAME] = dockerFuncObj.ContainerName
				cmdMap[commons.DOCKER_CMD] = dockerFuncObj.Cmd
				cmdMap[commons.PARAMETER] = dockerFuncObj.Param
				cmdMap[commons.SEQ] = strconv.Itoa(totalExecCount)
				wgTime.Add(1)
				batchExecCount++
				totalExecCount++

				cmdChan <- cmdMap
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

func cleanUpDocker() {
	commons.PrintToStdOutOnVerbose("Cleaning up created containers during the experiment!")
	for container, prevCmd := range containerPrevCmdMap {
		if prevCmd != commons.CONT_CMD_REMOVE {
			ExecCmd([]string{"stop", container})
			ExecCmd([]string{"rm", container})
		}
	}

	commons.PrintToStdOutOnVerbose("Clean up completed!")
}

/* execute single openwhisk cli command with argsArr arguments */
func ExecCmd(argsArr []string) string {
	var buffer bytes.Buffer
	buffer.WriteString("docker container ")

	for i := 0; i < len(argsArr); i++ {
		buffer.WriteString(argsArr[i])
		buffer.WriteString(" ")
	}

	args := strings.TrimSpace(buffer.String())
	commons.PrintToStdOutOnDebug(args)

	cmdOut, err := exec.Command("/bin/sh", "-c", args).CombinedOutput()
	if err != nil {
		panic(fmt.Errorf("Docker error - %s", cmdOut))
	}

	return strings.Trim(string(cmdOut), " \n")
}

func processResult(resultMap map[string]string) {
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

func invokeCommand() {
	for cmdMap := range cmdChan {
		containerName := cmdMap[commons.CONTAINER_NAME]
		dockerCmd := cmdMap[commons.DOCKER_CMD]
		param := cmdMap[commons.PARAMETER]

		paramArr := []string{dockerCmd}
		if dockerCmd == commons.CONT_CMD_CREATE || dockerCmd == commons.CONT_CMD_RUN {
			paramArr = append(paramArr, "--name="+containerName)
		} else {
			paramArr = append(paramArr, containerName)
		}

		if param != "" {
			paramArr = append(paramArr, param)
		}

		counterMtx.Lock()

		containerPrevCmd, ok := containerPrevCmdMap[containerName]
		if !ok {
			containerPrevCmd = commons.CONT_CMD_REMOVE
		}

		allowedCmds := dockerGraphMap[containerPrevCmd].Followers
		if !commons.ValueInSlice(dockerCmd, allowedCmds) {
			panic("Docker Error: Cannot run the command - " + dockerCmd + " as docker's previous command is " + containerPrevCmd)
		}

		counterMtx.Unlock()

		start := time.Now().UnixNano()
		execResult := ExecCmd(paramArr)
		end := time.Now().UnixNano()
		elapsed := (end - start) / 1000000 /* nano to milli */

		counterMtx.Lock()
		containerPrevCmdMap[containerName] = dockerCmd
		counterMtx.Unlock()

		resultMap := commons.CopyMap(cmdMap)
		resultMap[commons.CONTAINER_NAME] = containerName
		resultMap[commons.DOCKER_CMD] = dockerCmd
		resultMap[commons.SUBMITTED_AT] = strconv.FormatInt(start, 10)

		resultMap[commons.ENDED_AT] = strconv.FormatInt(end, 10)
		resultMap[commons.ELAPSED_TIME] = strconv.FormatInt(elapsed, 10)
		processResult(resultMap)
		wgTime.Done()

		if strings.HasPrefix(execResult, "error") {
			panic(fmt.Errorf("Error during execution - %s", execResult))
		}
	}
}
