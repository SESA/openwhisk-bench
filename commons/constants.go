package commons

const (
	// Common Constants
	BATCH                    = "Batch"
	PARAMETER                = "Parameter"
	SEQ                      = "Seq"
	ELAPSED_TIME             = "ElapsedTime"
	ELAPSED_TIME_SINCE_START = "ElapsedTimeSinceStart"
	SUBMITTED_AT             = "StartTime"
	ENDED_AT                 = "EndTime"
	EXEC_RATE                = "ExecRate"
	CONCURRENCY_FACTOR       = "ConcurrencyFactor"
	RECEIVED_BYTES           = "BytesReceived"
	TRANSMITTED_BYTES        = "BytesTransmitted"

	OPEN_WHISK_CONCURRENCY_FACTOR = 24

	// Open Whisk Contants
	USER_ID     = "UserID"
	USER_AUTH   = "UserAuth"
	FUNCTION_ID = "FunctionID"
	CMD_RESULT  = "ActivationId, WaitTime, InitTime, RunTime"
	CMD_STATUS  = "CmdStatus"

	// Docker Contants
	CONTAINER_NAME = "ContainerName"
	DOCKER_CMD     = "DockerCmd"

	CONT_CMD_CREATE = "create"
	CONT_CMD_REMOVE = "rm"
	CONT_CMD_RUN    = "run"
)
