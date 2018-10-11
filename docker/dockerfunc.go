package docker

import (
	"../commons"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"strconv"
	"strings"
)

var dockerGraphMap = make(map[string]DockerGraph)

type DockerFuncs struct {
	Seq           int
	ContainerName string
	Cmd           string
	Param         string
}

func (obj DockerFuncs) String() string {
	return "DockerFuncs: Seq - " + strconv.Itoa(obj.Seq) + ", ContainerName - " + obj.ContainerName + ", Cmd - " + obj.Cmd
}

type DockerGraph struct {
	ID        int      `yaml:"id"`
	Followers []string `yaml:"followers"`
}

func (obj DockerGraph) String() string {
	return "DockerGraph: ID - " + strconv.Itoa(obj.ID) + ", Followers - " + strings.Join(obj.Followers, ",")
}

func createDockerFuncsObj(contents []string) DockerFuncs {
	if len(contents) != 3 && len(contents) != 4 {
		panic(fmt.Errorf("Invalid Content Length - %d", len(contents)))
	}

	dockerFuncObj := DockerFuncs{
		Seq:           commons.GetIntFromStr(contents[0]),
		ContainerName: contents[1],
		Cmd:           contents[2],
	}

	if len(contents) == 4 {
		dockerFuncObj.Param = contents[3]
	}

	return dockerFuncObj
}

func parseYAML() {
	yamlFile, err := ioutil.ReadFile("docker/docker-life-cycle.yaml")
	if err != nil {
		panic(fmt.Errorf("yamlFile.Get err   #%v ", err))
	}

	err = yaml.Unmarshal([]byte(yamlFile), &dockerGraphMap)
	if err != nil {
		panic(fmt.Errorf("Unmarshal: %v", err))
	}
}

func getContainerStatusFromCommand(dockerCmd string) int {
	dockerGraph := dockerGraphMap[dockerCmd]
	return dockerGraph.ID
}
