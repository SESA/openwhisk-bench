package commons

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var newLineRegex = regexp.MustCompile(`\r?\n`)
var exists struct{}
var errorMsgsToSkip = map[string]struct{}{
	"resource already exists":  exists,
	"request timed out":        exists,
	"Document update conflict": exists,
	"but the request has not yet finished": exists,
}

var Debug bool
var WriteToFile bool
var RateLimit float64
var ConcurrencyFactor int
var RunForever = false
var Verbose = true
var OutputFileWriter os.File

func GetIntFromStr(strVal string) int {
	intVal, err := strconv.Atoi(strVal)
	if err != nil {
		panic(err)
	}

	return intVal
}

func delFromSlice(slice []interface{}, idxToDelete int) []interface{} {
	return append(slice[:idxToDelete], slice[idxToDelete+1:]...)
}

func CopyMap(mapToBeCopied map[string]string) map[string]string {
	targetMap := make(map[string]string)

	for key, value := range mapToBeCopied {
		targetMap[key] = value
	}

	return targetMap
}

func WriteMapToFile(writeMap map[string]string, printOrder []string) {
	printTxt := WriteMapToOut(writeMap, printOrder) + "\n"
	OutputFileWriter.WriteString(printTxt)
}

func WriteMapToOut(writeMap map[string]string, printOrder []string) string {
	var buffer bytes.Buffer

	for _, key := range printOrder {
		if ConcurrencyFactor != 1 && (key == RECEIVED_BYTES || key == TRANSMITTED_BYTES) {
			continue
		}

		if buffer.Len() > 0 {
			buffer.WriteString(", ")
		}

		buffer.WriteString(writeMap[key])
	}

	printTxt := strings.TrimSpace(buffer.String())
	PrintToStdOutOnVerbose(printTxt)
	return printTxt
}

func PrintHeader(printOrder []string, outputFilePath string) {
	var buffer bytes.Buffer

	for i := 0; i < len(printOrder); i++ {
		if ConcurrencyFactor != 1 && (printOrder[i] == RECEIVED_BYTES || printOrder[i] == TRANSMITTED_BYTES) {
			continue
		}

		delimiter := ", "
		if i == len(printOrder)-1 {
			delimiter = "\n"
		}

		if outputFilePath != "" {
			OutputFileWriter.WriteString(printOrder[i] + delimiter)
		}
		buffer.WriteString(printOrder[i] + delimiter)
	}

	printTxt := strings.TrimSpace(buffer.String())
	PrintToStdOutOnVerbose(printTxt)
}

func GenerateOutputFileName() string {
	executionTime := time.Now()
	outputFileName := "cmds_" + strconv.Itoa(executionTime.Year()) + "_" + executionTime.Month().String() + "_" + strconv.Itoa(executionTime.Day()) + "_" + strconv.Itoa(executionTime.Hour()) + "_" + strconv.Itoa(executionTime.Minute()) + "_" + strconv.Itoa(executionTime.Second()) + "_output.csv"
	return outputFileName
}

func CreateOutputFile(inputFilePath string) os.File {
	fileWriter, err := os.OpenFile(inputFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(fmt.Errorf("Cannot create file - %s", err))
	}

	PrintToStdOutOnVerbose("Writing output to " + inputFilePath)
	return *fileWriter
}

func PrintToStdOutOnVerbose(printTxt string) {
	if Verbose {
		fmt.Println(printTxt)
	}
}

func PrintToStdOutOnDebug(printTxt string) {
	if Debug {
		fmt.Println(printTxt)
	}
}

func shouldPanic(output string) bool {
	if len(strings.Split(output, ", ")) == 4 {
		return false
	}

	for msg := range errorMsgsToSkip {
		if strings.Contains(output, msg) {
			return false
		}
	}

	return true
}

func ParseJsonResponse(jsonStr string) (string, string) {
	jsonStr = newLineRegex.ReplaceAllString(jsonStr, " ")
	var jsonResp map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &jsonResp)
	if err != nil {
		fmt.Println("-----\n" + jsonStr + "\n-----")
		panic(err)
	}

	status := jsonResp["status"].(string)
	output := jsonResp["output"].(string)
	if status == "ERROR" && shouldPanic(output) {
		panic(fmt.Errorf("Bash error - %s", output))
	}

	if strings.Contains(output, "but the request has not yet finished") {
		output = "0, 0, 0, 0"
	}

	return status, output
}

func GetNetworkUsage() []int64 {
	cmdOut, err := exec.Command("ifconfig", "eno1").Output()
	if err != nil {
		panic(fmt.Errorf("Docker error - %s", err))
	}

	cmdOutStr := string(cmdOut)
	re := regexp.MustCompile("bytes:(.*?) \\(")
	lines := strings.Split(cmdOutStr, "\n")
	var networkData []int64

	for line := range lines {
		if !strings.Contains(lines[line], "bytes") {
			continue
		}

		match := re.FindAllStringSubmatch(lines[line], -1)
		for _, i := range match {
			val, err := strconv.ParseInt(i[1], 10, 64)
			if err != nil {
				panic(fmt.Errorf("Docker error - %s", err))
			}
			networkData = append(networkData, val)
		}
	}

	return networkData
}

func ValueInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
