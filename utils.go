package main

import (
	"strconv"
	"strings"
	"bytes"
	"fmt"
		"path/filepath"
	"time"
	"os"
	"log"
)

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

func writeMapToFile(fileName os.File, writeMap  map[string]string, printOrder []string) {
	var buffer bytes.Buffer

	for _, key := range printOrder {
		if buffer.Len() > 0 {
			buffer.WriteString(", ")
		}

		buffer.WriteString(writeMap[key])
	}

	printTxt := strings.TrimSpace(buffer.String())
	fmt.Println(printTxt) 
	printTxt += "\n"

	fileName.WriteString(printTxt)
}

func writeMapToOut(writeMap  map[string]string, printOrder []string) {
	var buffer bytes.Buffer

	for _, key := range printOrder {
		if buffer.Len() > 0 {
			buffer.WriteString(", ")
		}

		buffer.WriteString(writeMap[key])
	}

	printTxt := strings.TrimSpace(buffer.String())
	fmt.Println(printTxt) 
}

func createOutputFile(inputFilePath string) os.File {
	extension := filepath.Ext(inputFilePath)
	inputFileName := filepath.Base(inputFilePath)
	inputFileName = inputFileName[0:len(inputFileName)-len(extension)]

	executionTime := time.Now()
	outputFileName := inputFileName + "_" + strconv.Itoa(executionTime.Year()) + "_" + executionTime.Month().String() + "_" + strconv.Itoa(executionTime.Day()) + "_" + strconv.Itoa(executionTime.Hour()) + "_" + strconv.Itoa(executionTime.Minute()) + "_" + strconv.Itoa(executionTime.Second()) + ".csv"

	fileWriter, err := os.OpenFile(outputFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Cannot create file", err)
	}

	fmt.Println("Writing output to " + outputFileName)
	return *fileWriter
}
