package main

import (
	"strconv"
	"strings"
	"bytes"
	"fmt"
	"time"
	"os"
	"log"
	"encoding/json"
	"regexp"
)

var newLineRegex = regexp.MustCompile(`\r?\n`)
var exists struct{}
var errorMsgsToSkip = map[string]struct{}{
	"resource already exists":  exists,
	"request timed out":        exists,
	"Document update conflict": exists,
}

func getIntFromStr(strVal string) int {
	intVal, err := strconv.Atoi(strVal)
	if err != nil {
		panic(err)
	}

	return intVal
}

func delFromSlice(slice []interface{}, idxToDelete int) []interface{} {
	return append(slice[:idxToDelete], slice[idxToDelete+1:]...)
}

func copyMap(mapToBeCopied map[string]string) map[string]string {
	targetMap := make(map[string]string)

	for key, value := range mapToBeCopied {
		targetMap[key] = value
	}

	return targetMap
}

func writeMapToFile(fileName os.File, writeMap map[string]string, printOrder []string) {
	printTxt := writeMapToOut(writeMap, printOrder) + "\n"
	fileName.WriteString(printTxt)
}

func writeMapToOut(writeMap map[string]string, printOrder []string) string {
	var buffer bytes.Buffer

	for _, key := range printOrder {
		if buffer.Len() > 0 {
			buffer.WriteString(", ")
		}

		buffer.WriteString(writeMap[key])
	}

	printTxt := strings.TrimSpace(buffer.String())
	fmt.Println(printTxt)
	return printTxt
}

func generateOutputFileName() string {
	executionTime := time.Now()
	outputFileName := "cmds_" + strconv.Itoa(executionTime.Year()) + "_" + executionTime.Month().String() + "_" + strconv.Itoa(executionTime.Day()) + "_" + strconv.Itoa(executionTime.Hour()) + "_" + strconv.Itoa(executionTime.Minute()) + "_" + strconv.Itoa(executionTime.Second()) + "_output.csv"
	return outputFileName
}

func createOutputFile(inputFilePath string) os.File {
	fileWriter, err := os.OpenFile(inputFilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Cannot create file", err)
	}

	fmt.Println("Writing output to " + inputFilePath)
	return *fileWriter
}

func printToStdOutOnVerbose(printTxt string) {
	if verbose {
		fmt.Println(printTxt)
	}
}

func printToStdOutOnDebug(printTxt string) {
	if debug {
		fmt.Println(printTxt)
	}
}

func shouldPanic(output string) bool {
	for msg := range errorMsgsToSkip {
		if strings.Contains(output, msg) {
			return false
		}
	}

	return true
}

func parseJsonResponse(jsonStr string) string {
	jsonStr = newLineRegex.ReplaceAllString(jsonStr, " ")
	var jsonResp map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &jsonResp)
	if err != nil {
		fmt.Println("-----\n" + jsonStr + "\n-----")
		panic(err)
	}

	output := jsonResp["output"].(string)
	if jsonResp["status"] == "FAIL" && shouldPanic(output) {
		panic(fmt.Errorf("Bash error - %s", output))
	}

	return output
}
