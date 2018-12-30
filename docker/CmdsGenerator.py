"""
Used to generate list of test commands for the Go Wrapper.

-F = Filename to store the generated commands. If not given, it'll auto-generate a file based on current time.
-s = Upper limit of the sequence number (First Column in our input file). Say, the given sequence number is 2, it'll
    generate commands and then assign each of them either to sequence number 0 or sequence number 1. The distribution
    of commands between the sequence numbers will be almost equal.
-c = Min & Max limit for containers(minValue-maxValue). In each sequence, a number between this bound will be choosen
    and container operations are done over them.
-cmds = List of allowed commands. This command will be intersected with followers of each command in the Panic Graph
    and a new list of commandVsFollowers map will be created. Only commands from this list will be generated.
-imgs = List of allowed images. For each container creation, one of these images will be used. If this flag is absent,
    then the Scala image (b7a4814ab2aa) will be used by default.

Note: For container range, if you want to give a fixed number instead of a range, then you can give the fixed number
as both lower and upper limit. For example, if you want to keep the number of containers in all sequence to 5, then
give the range as "-c 5-5"
"""

import argparse
import collections
import os
import random
import time

import yaml

parser = argparse.ArgumentParser()
parser.add_argument("-F", "--filename", help="Filename to store")
parser.add_argument("-s", "--sequence", help="Sequence max limit")
parser.add_argument("-c", "--containers", help="No.of containers (lower-upper bound) to generate in each sequence")
parser.add_argument("-cmds", "--allowedCommands", help="List of allowed commands separated by comma. If list is "
                                                       "empty, then all commands are allowed.")
parser.add_argument("-imgs", "--allowedImages", help="List of images to use for create separated by comma. Default "
                                                     "image is b7a4814ab2aa (openwhisk/scala)")
parser.add_argument("-r", "--repetition", help="No. of repetition ")

CMD_CREATE = "create"
CMD_RUN = "run"
CMD_EXEC = "exec"
CMD_START = "start"
CMD_STOP = "stop"
CMD_PAUSE = "pause"
CMD_UNPAUSE = "unpause"
CMD_REMOVE = "rm"

commandsVsFollowers = {}

allowedCommandsVsFollowers = {}


def parseYAML():
    with open("docker/docker-life-cycle.yaml", 'r') as fread:
        try:
            global commandsVsFollowers
            commandsVsFollowers = yaml.load(fread)
        except yaml.YAMLError as exc:
            print(exc)


def getDockerCmd(containerStatus):
    allowedStatuses = allowedCommandsVsFollowers.get(containerStatus)
    return random.choice(allowedStatuses)


def generateCommands(seqNo, noOfContainersMin, noOfContainersMax, imgs):
    cmds_map = {}
    container_status_map = {}

    for seq in range(seqNo):
        no_of_containers = random.randint(noOfContainersMin, noOfContainersMax)
        for cont_id in range(no_of_containers):
            cont_name = "cont_" + str(cont_id)
            cont_status = container_status_map.get(cont_name) or CMD_REMOVE
            docker_cmd = getDockerCmd(cont_status)
            cmd_map = {"container": cont_name, "cmd": docker_cmd}

            if docker_cmd == CMD_RUN:
                img = random.choice(imgs)
                cmd_map["param"] = "-d --cpu-shares 0 --memory 256m --oom-kill-disable --network bridge -e __OW_API_HOST=https://0.0.0.0:443 openwhisk/nodejs4action:latest"

            container_status_map[cont_name] = docker_cmd
            cmds_list = cmds_map.get(seq) or []
            cmds_list.append(cmd_map)
            cmds_map[seq] = cmds_list

    ordered_cmds_map = collections.OrderedDict(sorted(cmds_map.items()))
    return ordered_cmds_map


def generateOutputFile():
    file_name = time.strftime("%Y_%B_%d_%H_%M_%S")
    file_name = "cmds_" + file_name + "_input.csv"
    return file_name


if __name__ == "__main__":
    parseYAML()

    args = parser.parse_args()

    fileName = args.filename or generateOutputFile()
    currDir = os.getcwd()
    if currDir != "docker" and not fileName.startswith("docker/"):
        fileName = "docker/" + fileName

    print(fileName)

    seqNo = args.sequence or 15
    seqNo = int(seqNo)

    noOfContainersMin, noOfContainersMax = 10, 20
    if args.containers is not None:
        noOfContainersMin = int(args.containers.split("-")[0])
        noOfContainersMax = int(args.containers.split("-")[1])

    userAllowedCommands = set(args.allowedCommands.split(",")) if args.allowedCommands is not None else None
    for cmd, details in commandsVsFollowers.items():
        followers = details["followers"]
        if userAllowedCommands is not None:
            allowedCommands = list(userAllowedCommands & set(followers))
        else:
            allowedCommands = followers
        allowedCommandsVsFollowers[cmd] = allowedCommands

    imgs = args.allowedImages or "b7a4814ab2aa"
    imgs = imgs.split(",")

    print("FileName: " + fileName, "SeqNo: " + str(seqNo),
          "NoOfContainers: " + str(noOfContainersMin) + "-" + str(noOfContainersMax),
          "AllowedCommands: " + ', '.join(allowedCommandsVsFollowers.keys()),
          "AllowedImages: " + ', '.join(imgs))

    repetition = args.repetition or 1
    repetition = int(repetition)

    cmds_map = generateCommands(seqNo, noOfContainersMin, noOfContainersMax, imgs)

    with open(fileName, "w") as fwrite:
        for key, val in cmds_map.items():
            for _ in range(repetition):
                for cmd_data in val:
                    fwrite.write(str(key) + ",")
                    fwrite.write(str(cmd_data.get("container")) + ",")
                    fwrite.write(str(cmd_data.get("cmd")))

                    if "param" in cmd_data:
                        fwrite.write("," + str(cmd_data.get("param")))

                    fwrite.write("\n")
