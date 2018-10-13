#!/bin/bash
######################################################################
#  Openwhisk Benchmark Trail Runner
#
######################################################################

export OWBENCH=${WSK_BENCH:=$PWD/openwhisk-bench}
export OWDEPLOY=${OW_DEPLOY:=$PWD/../ow-deploy.sh}
export OWWRAP=${OW_WRAP:="go run *.go -q "}

export TRIALDATADIR=${TRIAL_DATA_DIR:=$PWD/data}

export CMDPREFIX=${CMD_PREFIX:=""}
export CMDPOSTFIX=${CMD_POSTFIX:="&> /dev/null"}

if [[ ! -z $DEBUG ]]; then set -x; fi

echo "Begin OpenWhisk Benchmark Trial"

function CMDR
{
  local CMD="$CMDPREFIX ${@} $CMDPOSTFIX"
  /bin/bash -c "$CMD"
}

function SingleTrial
{
  local START_TIME=$SECONDS
  local TPATH=$1
  FILES=$(/bin/bash -c "ls $TPATH | grep csv")
  
  echo "Batch files: $FILES"

  TrialInit

  echo "$SECONDS: Booting OpenWhisk..."
  CMDR $OWDEPLOY Boot 

  LAST_FILE=""
  for f in $FILES
  do
  	if [ ! -z $LAST_FILE ]
  	then
  	        echo "$SECONDS: Processing batch file $LAST_FILE"
   		DoRun $TPATH$LAST_FILE; 
   		echo "$SECONDS: Rebooting OpenWhisk..."
   		CMDR $OWDEPLOY Reboot 
  	fi
  	LAST_FILE=$f
  done
  if [ ! -z $LAST_FILE ]
  then
        echo "$SECONDS: Process final batch file $LAST_FILE"
   	DoRun $TPATH$LAST_FILE; 
  fi
  echo "$SECONDS: Shutting down OpenWhisk..."
  CMDR $OWDEPLOY Shutdown 
  local ELAPSED_TIME=$(($SECONDS - $START_TIME))
  echo "$SECONDS: Finished Trial in $ELAPSED_TIME seconds"
}

function TrialInit
{
  if [[ -z $TRIALID ]]; then
    export TRIALID=$(/bin/date +%d-%m-%y-%H:%M)
    export TRIALPATH=$TRIALDATADIR/$TRIALID
  fi
  CMDR mkdir -p $TRIALPATH 
}

function DoRun 
{
  local NAME=$(/bin/bash -c "echo $1 | /usr/bin/cut -d '/' -f 2")
  if [[ -n $SEUSS ]]; then
    FN="$TRIALPATH/seuss_$NAME"
  else 
    FN="$TRIALPATH/linux_$NAME"
  fi
  CMDR touch $FN
  echo -e "\t $FN"
  local START_TIME=$SECONDS
  CMDR $OWWRAP --create --writeToFile --fileName $FN execFile $1
  local ELAPSED_TIME=$(($SECONDS - $START_TIME))
  echo -e "\tRun finished in $ELAPSED_TIME seconds"
}

#######################################################################

usage()
{
  local func=$1
  if [[ -z $func ]]
  then
     echo "USAGE:  ${0##*/} func args" >&2
     grep '^function' $0
  fi
}

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
