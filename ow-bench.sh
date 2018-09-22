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
    seed=$1

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
        mkdir -p $TMPDIR
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

	OUTPUT=$(bash -c "$WSKCLI -i --apihost $WSKHOST action invoke -b $@ | tail -n +2" 2>&1)

    if [[ $OUTPUT == error* ]]; then
        echo "$OUTPUT" > /dev/stderr
        echo -1, -1, -1, -1
    else
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
        duration_t=`expr $run_t - $init_t`

        echo $aid, $wait_t, $init_t, $duration_t 
    fi
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
