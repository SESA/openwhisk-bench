"""
Used to generate list of test commands for the Go Wrapper.

-F = Filename to store the generated commands. If not given, it'll auto-generate a file based on current time.
-s = Upper limit of the sequence number (First Column in our input file). Say, the given sequence number is 2, it'll
    generate commands and then assign each of them either to sequence number 0 or sequence number 1. The distribution
    of commands between the sequence numbers will be almost equal.
-u = Upper limit for the user id. If the number is 10, it'll generate 10 different users and assign generated functions
    to each of them randomly.
-f = Min & Max limit for functions(minValue-maxValue). For each user, first it'll pick a random number in this given
    range. Then, it'll generate that number of functions for that user each with a different id. Say, the given function
    range is 5-10 and the picked random number is 6, then 6 functions with different id will be generated for the user.
-e = Min & Max limit for number of executions(minValue-maxValue). For each function it generates for each user, it picks
    a random number in this range and that will be the number of executions for that function and user.

Once number of executions is decided, then it assigns a seq number randomly and create a row with list of generated
parameters. Once all such rows are generated, it sorts the rows based on the increasing order of the seq number and
then write them to the given/generated file name.

Note: For both function range & execution range, if you want to give a fixed number instead of a range, then you can
give the fixed number as both lower and upper limit. For example, if you want to keep the number of executions of all
commands to 5, then give the range as "-e 5-5"
"""

import argparse
import collections
import random
import time

parser = argparse.ArgumentParser()
parser.add_argument("-F", "--filename", help="Filename to store")
parser.add_argument("-s", "--sequence", help="Sequence max limit")
parser.add_argument("-u", "--users", help="No. of users to generate")
parser.add_argument("-f", "--functions", help="No.of functions (lower-upper bound) to generate for each user")
parser.add_argument("-e", "--execution", help="No. of times to execute (lower-upper bound) each function")


def generateCommands(seqNo, noOfUsers, noOfFunctionsMin, noOfFunctionsMax, noOfExecutionMin, noOfExecutionMax):
    cmds_map = {}

    for _ in range(noOfUsers):
        user_id = random.randrange(noOfUsers)
        no_of_func = random.randint(noOfFunctionsMin, noOfFunctionsMax)
        for _ in range(no_of_func):
            seq_no = random.randrange(seqNo)
            func_id = random.randint(noOfFunctionsMin, noOfFunctionsMax)
            no_of_exec = random.randint(noOfExecutionMin, noOfExecutionMax)

            cmd_map = {"user": user_id, "func": func_id, "exec": no_of_exec}
            cmds_list = cmds_map.get(seq_no) or []
            cmds_list.append(cmd_map)
            cmds_map[seq_no] = cmds_list

    ordered_cmds_map = collections.OrderedDict(sorted(cmds_map.items()))
    return ordered_cmds_map


def generateOutputFile():
    file_name = time.strftime("%Y_%B_%d_%H_%M_%S")
    file_name = "cmds_" + file_name + "_input.csv"
    return file_name


if __name__ == "__main__":
    args = parser.parse_args()

    fileName = args.filename or generateOutputFile()
    print(fileName)

    seqNo = args.sequence or 15
    seqNo = int(seqNo)

    noOfUsers = args.users or 10
    noOfUsers = int(noOfUsers)

    noOfFunctionsMin, noOfFunctionsMax = 10, 20
    if args.functions is not None:
        noOfFunctionsMin = int(args.functions.split("-")[0])
        noOfFunctionsMax = int(args.functions.split("-")[1])

    noOfExecutionMin, noOfExecutionMax = 5, 20
    if args.execution is not None:
        noOfExecutionMin = int(args.execution.split("-")[0])
        noOfExecutionMax = int(args.execution.split("-")[1])

    print("FileName: " + fileName, "SeqNo: " + str(seqNo), "NoOfUsers: " + str(noOfUsers),
          "NoOfFunctions: " + str(noOfFunctionsMin) + "-" + str(noOfFunctionsMax),
          "NoOfExecutions: " + str(noOfExecutionMin) + "-" + str(noOfExecutionMax))

    cmds_map = generateCommands(seqNo, noOfUsers, noOfFunctionsMin, noOfFunctionsMax, noOfExecutionMin,
                                noOfExecutionMax)

    with open(fileName, "w") as fwrite:
        for key, val in cmds_map.items():
            for cmd_data in val:
                fwrite.write(str(key) + ",")
                fwrite.write(str(cmd_data.get("user")) + ",")
                fwrite.write(str(cmd_data.get("func")) + ",")
                fwrite.write(str(cmd_data.get("exec")))
                fwrite.write("\n")
