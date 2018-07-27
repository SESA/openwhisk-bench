package main

import (
		"strconv"
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