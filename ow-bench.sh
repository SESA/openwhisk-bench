#! /bin/bash
#
#  Benchmarking & experimentation tool for Apache OpenWhisk
#
#  This files provides a collectiion of helper scripts for interacting
#	   with a configured OpenWhisk deployment.
#
#  This script requies the `wsk` and `wskadmin` binaries defined in your PATH:
#			e.g., export PATH=$PATH:$HOME/incubator-openwhisk/bin
#  Or, you can manually set the WSK_CLI and WSK_ADMIN environment variables
#
#  The following utilities are also required: bc, jq
#
#	 USAGE: ow-bench.sh cmd args
#
#  The following arguments can be used on any command:
#			COUNT = amount of repete cmds to run
#			DELAY = time to sleep between runs (poll if DELAY is set but COUNT is not)
#			CLEAR = clear the screen between runs
#			DEBUG = print full commands
#set -x
export WSKCLI=${WSK_CLI:=wsk}
export WSKADMIN=${WSK_ADMIN:=wskadmin}
export WSKLOG=${WSK_INVOKER_LOG:=/tmp/wsklogs/invoker0/invoker0_logs.log}
export TMPDIR=${TMP_DIR:=/tmp}
export WSKUSER=${WSK_USER:=guest}
export WSKAUTH=${WSK_AUTH:=23bc46b1-71f6-4ed5-8c54-816aa4f8c502:123zO3xZCLrMN6v2BKK1dXYFpXlPkccOFqm12CdAsMgRU4VrNZ9lyGVCGuMDGIwP}
export WSKHOST=${WSK_HOST:=http://172.17.0.1:10001}  # Bypass API gateway, send requests direct to the Controller 
export DEBUG=${DEBUG:=}
export CLEAR=${CLEAR:=}
export COUNT=${COUNT:=}
export WAIT=${WAIT:=}
if [[ -n $DEBUG ]]; then set -x; fi
if [[ -n $WAIT ]]; then
  if [[ $(bc -l <<< "0 < $WAIT") -eq 1 ]]; then
    echo "Poll frequency set at $WAIT seconds"
  else
    WAIT=""
  fi
fi

usage()
{
  local func=$1
  if [[ -z $func ]]
  then
     echo "USAGE:  ${0##*/} func args" >&2
     grep '^function' $0
  else                   ## Todo
     case "$func" in
         'fooBar')
            echo "USAGE: ${0##*/} fooBar " >&2
            echo "     -f foo   : Set foo in fooBar" >&2
            echo "     -b bar   : Set bar in fooBar" >&2
            ;;
          *)
            usage
            ;;
     esac
  fi
}

####################################################

# Wrapper around the wsk command
function wskCli
{
  cmd="$WSKCLI -i --apihost $WSKHOST $@"
  bash -c "$cmd"
}




# Wrapper around the wskadmin command
function wskAdmin
{
  cmd="$WSKADMIN $@"
  bash -c "$cmd"
}




# Wrapper around go benchmark tool
function runTest
{
  local test_file=$1
  if [[ -z $test_file ]];
  then
    echo "Error: Cannot Run Tests; Test File Path Needed"
    return
  fi

  if [[ "$2" = "--stats" ]];
  then
    if [[ -f temp.csv ]];
    then
      rm temp.csv
    fi
    if [[ -f stats.csv ]];
    then
      rm stats.csv
    fi

    go run *.go --create execFile $test_file >> temp.csv
    local total_time="$(cat temp.csv | awk '/Total_Job_Time/ {print}')"

    cat temp.csv | tail -n+9 | head -n -3 >> stats.csv
    local numInvocations="$(cat stats.csv | wc -l)"
    local total_wait="$(cat stats.csv | awk -F "\"*,\"*" '{print $5}' | awk '{print $1}' | paste -sd+ | bc)"
    local total_init="$(cat stats.csv | awk -F "\"*,\"*" '{print $5}' | awk '{print $2}' | paste -sd+ | bc)"
    local total_duration="$(cat stats.csv | awk -F "\"*,\"*" '{print $5}' | awk '{print $3}' | paste -sd+ | bc)"

    # averages
    local avg_wait="$((total_wait / numInvocations))"
    local avg_init="$((total_init / numInvocations))"
    local avg_duration="$((total_duration / numInvocations))"

    # min, max
    local min_wait="$(cat stats.csv | awk -F "\"*,\"*" '{print $5}' | awk '{print $1}' | sort | head -1)"
    local max_wait="$(cat stats.csv | awk -F "\"*,\"*" '{print $5}' | awk '{print $1}' | sort -rn | head -1)"
    local min_init="$(cat stats.csv | awk -F "\"*,\"*" '{print $5}' | awk '{print $2}' | sort | head -1)"
    local max_init="$(cat stats.csv | awk -F "\"*,\"*" '{print $5}' | awk '{print $2}' | sort -rn | head -1)"
    local min_duration="$(cat stats.csv | awk -F "\"*,\"*" '{print $5}' | awk '{print $3}' | sort | head -1)"
    local max_duration="$(cat stats.csv | awk -F "\"*,\"*" '{print $5}' | awk '{print $3}' | sort -rn | head -1)"

    # stddev
    local wait_stdev="$(cat stats.csv | awk -F "\"*,\"*" '{print $5}' | awk '{print $1}' | awk '{sum+=$1; sumsq+=$1*$1}END{print int(sqrt(sumsq/NR - (sum/NR)**2))}')"
    local init_stdev="$(cat stats.csv | awk -F "\"*,\"*" '{print $5}' | awk '{print $2}' | awk '{sum+=$1; sumsq+=$1*$1}END{print int(sqrt(sumsq/NR - (sum/NR)**2))}')"
    local duration_stdev="$(cat stats.csv | awk -F "\"*,\"*" '{print $5}' | awk '{print $3}' | awk '{sum+=$1; sumsq+=$1*$1}END{print int(sqrt(sumsq/NR - (sum/NR)**2))}')"

    echo "See Full Output in Temporary File: stats.csv"
    echo ""
    echo "Total_Wait_Time: $total_wait"
    echo "Total_Init_Time: $total_init"
    echo "Total_Duration_Time: $total_duration"
    echo ""
    echo "Average_Wait_Time: $avg_wait"
    echo "Average_Init_Time: $avg_init"
    echo "Average_Duration_Time: $avg_duration"
    echo ""
    echo "Min_Wait_Time: $min_wait"
    echo "Max_Wait_Time: $max_wait"
    echo "Stdev_Wait: $wait_stdev"
    echo ""
    echo "Min_Init_Time: $min_init"
    echo "Max_Init_Time: $max_init"
    echo "Stdev_Init: $init_stdev"
    echo ""
    echo "Min_Duration_Time: $min_duration"
    echo "Max_Duration_Time: $max_duration"
    echo "Stdev_Duration: $duration_stdev"
    echo ""
    echo "$total_time"

  else
    go run *.go --create execFile $test_file
  fi
}



function countContainers
{
  extra=""
  if [[ $# -gt 0 ]]; then
    for i in "$@"; do
      extra="$extra | grep $i "
    done
  fi
  cmd="docker ps | grep whisk $extra | wc -l"
  bash -c "$cmd"
}

function showContainers
{
  extra=""
  if [[ $# -gt 0 ]]; then
    for i in "$@"; do
      extra="$extra | grep $i "
    done
  fi
  cmd="docker ps $extra "
  bash -c "$cmd"
}




function showStarts
{
  extra=""
  if [[ $# -gt 0 ]]; then
    for i in "$@"; do
      extra="$extra | grep $i "
    done
  fi
  cmd="grep containerState $WSKLOG 2> /dev/null $extra | grep -v invokerHealth "
  bash -c "$cmd"
}

function countStarts
{
  echo -e "cold start:\t\t" $( showStarts cold | wc -l )
  echo -e "prewarm start:\t\t" $( showStarts prewarm | wc -l )
  echo -e "recreated:\t\t" $( showStarts recreated  | wc -l )
  echo -e "warm start:\t\t" $( showStarts warm | grep -v prewarm  | wc -l )
  echo -e "total start:\t\t" $( showStarts | wc -l )
}

function countAll
{
  echo -e "containers:\t\t" $(countContainers nodejs) 
  countStarts
}



function createUser
{
    if [[ $1 != user* ]]; then
	    seed=user_$1
	else
	    seed=$1
	fi;

	output=`wskAdmin user create $seed`

	if [ "$output" = "Namespace already exists" ]; then
		output=`getUserAuth $seed`
	fi;

	echo $seed $output
}

function randomUser
{
	createUser $RANDOM
}




function createFunction
{
    if [ "$#" -lt 2 ];
    then
        echo "Error: Too Few Parameters to createFunction"
        return
    fi
 
    local user_name=$1
    local user_auth=$(wskadmin user get $user_name)
    local action_name=$2

    local action_func=$3
    if [ -z "$action_func" ];
    then
        action_func="$TMPDIR/wsk_fun_$action_name.js"
        touch $action_func
        echo "function main() { return {payload: 'RANDOM $seed'}; }" > $action_func
    fi

    wskCli --auth $user_auth action create $action_name $action_func > /dev/null

    if [ $? -eq 0 ]; then
	    echo $action_name
    fi
}




function randomFunction
{
	createFunction guest $RANDOM 
}




function updateFunction
{
    if [ "$#" -lt 3 ];
    then
        echo "Error: Too Few Parameters to updateFunction"
        return
    fi
 
    local user_name=$1
    if [ -z "$user_name" ];
    then
        echo "Error: Cannot Create User Function; Need User Name"
        return
    fi
    local user_auth=$(wskadmin user get $user_name)

    local action_name=$2
    if [ -z "$action_name" ];
    then
        echo "Error: Cannot Update Guest Function; Need User Function Name"
        return
    fi

    local action_func=$3
    if [ -z "$action_func" ];
    then
        echo "Error: Cannot Update Guest Function; Need User Function Implementation"
        return
    fi
    
    wsk -i --apihost $WSKHOST --auth $user_auth action update $action_name $action_func
}




function getUserAuth {
	local user=$1
	if [ -z "$user" ]; then
		user=$WSKUSER
	fi
	echo $( wskAdmin user get $user )
}

# getInvokeTime
# Invoke function (blocking)
#	Returns <wait_time> <init_time> <run_time>
function getInvokeTime
{
	init_t=0
	wait_t=0
	run_t=0
    	OUTPUT=$(bash -c "$WSKCLI -i  --apihost $WSKHOST action invoke -b $@ | tail -n +2")

    if [ -z "$OUTPUT" ]; then
        OUTPUT="Threshold Reached Warning!"
				#TODO: RETURN ERROR,STOP EXPERIMENT
    fi

	len=$(echo $OUTPUT | jq -r '.annotations | length')
	run_t=$( echo $OUTPUT | jq -r '.duration' )

	if [[ $len -eq 4 ]]; then # WARM/HOT START
		wait_t=$( echo $OUTPUT | jq -r '.annotations' | jq -r '.[3]' | jq -r '.value' )
	elif [[ $len -eq 5 ]]; then #COLD START
		wait_t=$( echo $OUTPUT | jq -r '.annotations' | jq -r '.[1]' | jq -r '.value' )
		init_t=$( echo $OUTPUT | jq -r '.annotations' | jq -r '.[4]' | jq -r '.value' )
	elif [[ $len -eq 2 ]]; then #SEUSS RETURN
		wait_t=$( echo $OUTPUT | jq -r '.annotations' | jq -r '.[0]' | jq -r '.value' )
		init_t=$( echo $OUTPUT | jq -r '.annotations' | jq -r '.[1]' | jq -r '.value' )
	fi

	aid=$( echo $OUTPUT | jq -r '.activationId' )
	
	final_run_t=`expr $run_t - $init_t`
	echo $wait_t $init_t $final_run_t $aid
}




function invokeFunction
{
    if [ "$1" = "--verbose" ];
    then
        user_name=$2
        set -- "${@:1:1}" "${@:3}"
    else
        user_name=$1
        shift
    fi

    local user_auth=$(wskadmin user get $user_name)

    invokeFunctionWithAuth $user_auth $@
}




function invokeFunctionWithAuth {
    if [ "$#" -lt 2 ];
    then
        echo "Error: Too Few Parameters to invokeFunction"
        return
    fi

    local user_auth=$1
    shift

    if [ "$1" = "--verbose" ];
    then
        verbosity=1
        shift
    else
        verbosity=0
    fi

    local action_name=$1
    shift

    local action_flags=$@
    if [ -z "$action_flags" ];
    then
        #echo "Invoking $action_name with no parameters."

        if [ $verbosity -eq 1 ];
        then
            wsk -i --apihost $WSKHOST --auth $user_auth action invoke -b $action_name
        else
            getInvokeTime "-u \"$user_auth\" $action_name"
        fi
    else
        if [ "$1" = "--param" ] || [ "$1" = "-p" ];
        then
            shift
            local action_params=$@
            #echo "Invoking $action_name with parameters: $action_params"

            if [ $verbosity -eq 1 ];
            then
                wsk -i --apihost $WSKHOST --auth $user_auth action invoke -b $action_name --param $action_params
            else
                getInvokeTime "-u \"$user_auth\" $action_name --param $action_params"
            fi
        else
            echo "Error: Invalid Flag"
            return
        fi
    fi
}




function deleteFunction
{
    if [ "$#" -lt 2 ];
    then
        echo "Error: Too Few Parameters to deleteFunction"
        return
    fi
 
    local user_name=$1
    if [ -z "$user_name" ];
    then
        echo "Error: Cannot Create User Function; Need User Name"
        return
    fi
    local user_auth=$(wskadmin user get $user_name)

    local action_name=$2
    if [ -z "$action_name" ];
    then
        echo "Error: Cannot Delete Guest Function; Need User Function Name"
        return
    fi

    wsk -i --apihost $WSKHOST --auth $user_auth action delete $action_name
}




function getFunction
{
    if [ "$#" -lt 2 ];
    then
        echo "Error: Too Few Parameters to getFunction"
        return
    fi
 
    local user_name=$1
    if [ -z "$user_name" ];
    then
        echo "Error: Cannot Create User Function; Need User Name"
        return
    fi
    local user_auth=$(wskadmin user get $user_name)

    local action_name=$2
    if [ -z "$action_name" ];
    then
        echo "Error: Cannot Get Guest Function Metadata; Need Guest Function Name"
        return
    fi

    wsk -i --apihost $WSKHOST --auth $user_auth action get $action_name
}




function listFunctions
{
    local user_name=$1
    if [ -z "$user_name" ];
    then
        echo "Error: Cannot List User Functions; Need User Name"
        return
    fi
    local user_auth=$(wskadmin user get $user_name)

    wsk -i --apihost $WSKHOST --auth $user_auth action list
}

####################################################

processargs()
{
  if [[ $# == 0 ]]
  then
    usage
    exit -1
  fi

  dofunc=$1
}

if [[ -n $WAIT ]]; then
   processargs "$@"
   shift
   while [ 1 ]; do
     if [[ -n $CLEAR ]]; then
       clear
     fi
     $dofunc "$@"
     sleep $WAIT
   done
elif [[ $COUNT -gt 0 ]]; then
  processargs "$@"
  shift
  for (( i=1; i<=$COUNT; i++ )); do
    if [[ -n $CLEAR ]]; then
      clear
    fi
    $dofunc "$@"
  done
else
  processargs "$@"
  shift
  if [[ -n $CLEAR ]]; then
    clear
  fi
  $dofunc "$@"
  exit $?
fi
