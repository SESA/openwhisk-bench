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
    print(seqNo, noOfUsers, noOfFunctionsMin, noOfFunctionsMax, noOfExecutionMin, noOfExecutionMax)
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
    # for key, val in ordered_cmds_map.items():
    #     print(key, val)
    return ordered_cmds_map


def generateOutputFile():
    file_name = time.strftime("%Y_%B_%d_%H_%M_%S")
    file_name = "cmds_" + file_name + "_input.csv"
    return file_name


if __name__== "__main__":
    args = parser.parse_args()

    fileName = args.filename or generateOutputFile()
    print(fileName)

    seqNo = int(args.sequence) or 15
    noOfUsers = int(args.users) or 10

    noOfFunctionsMin, noOfFunctionsMax = 10, 20
    if args.functions is not None:
        noOfFunctionsMin = int(args.functions.split("-")[0])
        noOfFunctionsMax = int(args.functions.split("-")[1])

    noOfExecutionMin, noOfExecutionMax = 5, 20
    if args.execution is not None:
        noOfExecutionMin = int(args.execution.split("-")[0])
        noOfExecutionMax = int(args.execution.split("-")[1])

    cmds_map = generateCommands(seqNo, noOfUsers, noOfFunctionsMin, noOfFunctionsMax, noOfExecutionMin, noOfExecutionMax)

    with open(fileName, "w") as fwrite:
        for key, val in cmds_map.items():
            for cmd_data in val:
                fwrite.write(str(key) + ",")
                fwrite.write(str(cmd_data.get("user")) + ",")
                fwrite.write(str(cmd_data.get("func")) + ",")
                fwrite.write(str(cmd_data.get("exec")))
                fwrite.write("\n")
