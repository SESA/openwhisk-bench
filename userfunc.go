package main

import (
	"fmt"
	"strconv"
)

type UserFuncs struct {
	Time               int
	UserID             string
	FunctionID         int
	NoOfTimesToExecute int
}

func (obj UserFuncs) String() string {
	return "UserFuncs: Time - " + strconv.Itoa(obj.Time) + ", UserID - " + obj.UserID + ", FunctionID - " + strconv.Itoa(obj.FunctionID) + ", NoOfTimeToExecute - " + strconv.Itoa(obj.NoOfTimesToExecute)
}

func createUserFuncsObj(contents []string) UserFuncs {
	if len(contents) != 4 {
		panic(fmt.Errorf("Invalid Content Length - %d", len(contents)))
	}

	return UserFuncs{
		Time:               getIntFromStr(contents[0]),
		UserID:             "user_" + contents[1],
		FunctionID:         getIntFromStr(contents[2]),
		NoOfTimesToExecute: getIntFromStr(contents[3]),
	}
}
