package openwhisk

import (
	"../commons"
	"fmt"
	"strconv"
)

type UserFuncs struct {
	Time               int
	UserID             string
	FunctionID         int
	Param              string
	NoOfTimesToExecute int
}

func (obj UserFuncs) String() string {
	return "UserFuncs: Time - " + strconv.Itoa(obj.Time) + ", UserID - " + obj.UserID + ", FunctionID - " + strconv.Itoa(obj.FunctionID) + ", NoOfTimeToExecute - " + strconv.Itoa(obj.NoOfTimesToExecute)
}

func createUserFuncsObj(contents []string) UserFuncs {
	if len(contents) != 4 && len(contents) != 5 {
		panic(fmt.Errorf("Invalid Content Length - %d, %+v", len(contents), contents))
	}

	userFuncObj := UserFuncs{
		Time: commons.GetIntFromStr(contents[0]),
		UserID: func() string {
			if _, err := strconv.Atoi(contents[1]); err == nil {
				return "user_" + contents[1]
			} else {
				return contents[1]
			}
		}(),
		FunctionID: commons.GetIntFromStr(contents[2]),
		NoOfTimesToExecute: commons.GetIntFromStr(func() string {
			if len(contents) == 4 {
				return contents[3]
			} else {
				return contents[4]
			}
		}()),
	}

	if len(contents) == 5 {
		userFuncObj.Param = contents[3]
	}

	return userFuncObj
}
