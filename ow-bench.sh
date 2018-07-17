#!/bin/bash
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

export WSKCLI=${WSK_CLI:=wsk}
export WSKADMIN=${WSK_ADMIN:=wskadmin}
export WSKLOG=${WSK_INVOKER_LOG:=/tmp/wsklogs/invoker0/invoker0_logs.log}
export TMPDIR=${TMP_DIR:=/tmp}
export WSKUSER=${WSK_USER:=guest} 
export WSKAUTH=${WSK_AUTH:=23bc46b1-71f6-4ed5-8c54-816aa4f8c502:123zO3xZCLrMN6v2BKK1dXYFpXlPkccOFqm12CdAsMgRU4VrNZ9lyGVCGuMDGIwP} 
export DEBUG=${DEBUG:=}
export CLEAR=${CLEAR:=}
export COUNT=${COUNT:=}
export WAIT=${WAIT:=}
if [[ -n $DEBUG ]]; then set -x; fi 
if [[ -n $WAIT ]]; then
  if [[ $(bc -l <<< "0 < $WAIT") -eq 1 ]]; then
    echo "Poll frequency set at $WAIT seconds"
  else
    set WAIT=""
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
  cmd="$WSKCLI -i $@"
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
  cmd="grep containerState $WSKLOG $extra "
  bash -c "$cmd"
}

function countStarts
{
  echo -e "cold:\t\t" $( showStarts cold | wc -l )
  echo -e "prewarm:\t" $( showStarts prewarm | wc -l )
  echo -e "warm:\t\t" $( showStarts warm | wc -l )
}

function randomUser
{
	seed=user_$RANDOM
	echo -n $seed "" 
  wskAdmin user create $seed 
}

function randomFunction
{
	local auth=$1
	if [ -z "$auth" ]; then
		auth=$WSKAUTH
	fi 
	seed=$RANDOM
	file="$TMPDIR/wsk_func_$RANDOM.js"
	touch $file
  cat << EOF >> $file
 function main() {
    return {payload: 'RANDOM $seed'};
 }
EOF
  wskCli -u $auth create $seed $file > /dev/null
  if [ $? -eq 0 ]; then
		echo $seed
	fi
	rm $file
}

function getUserAuth {
	local user=$1
	if [ -z "$user" ]; then
		user=$WSKUSER
	fi 
	echo -n $( wskAdmin user get $user ) 
}

# getInvokeTime 
# Invoke function (blocking)
#	Returns <wait_time> <init_time> <run_time> 
function getInvokeTime
{
	init_t=0
	wait_t=0
	run_t=0
  OUTPUT=$(bash -c "$WSKCLI -i action invoke -b $@ | tail -n +2")
	len=$(echo $OUTPUT | jq -r '.annotations | length') 
	run_t=$( echo $OUTPUT | jq -r '.duration' )
	if [[ $len -eq 4 ]]; then # WARM/HOT START 
		wait_t=$( echo $OUTPUT | jq -r '.annotations' | jq -r '.[3]' | jq -r '.value' )
	elif [[ $len -eq 5 ]]; then #COLD START
		wait_t=$( echo $OUTPUT | jq -r '.annotations' | jq -r '.[1]' | jq -r '.value' )
		init_t=$( echo $OUTPUT | jq -r '.annotations' | jq -r '.[4]' | jq -r '.value' )
	fi
	echo $wait_t $init_t $run_t 
}

# invokeFunction <function> <user>
function invokeFunction {
	local function=$1
	local user=$2
	if [ -z "$user" ]; then
		user=$WSKUSER
	fi 
  getInvokeTime "-u $( getUserAuth $user ) $function"
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
