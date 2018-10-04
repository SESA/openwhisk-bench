#!/bin/bash
######################################################################
#  Openwhisk Trail Runner
#
######################################################################

export OWBENCH=${WSK_BENCH:=$PWD/openwhisk-bench}
export OWDEPLOY=${OW_DEPLOY:=$PWD/../ow-deploy.sh}
export OWWRAP=${OW_WRAP:="go run *.go -q"}

export TRIALDATADIR=${TRIAL_DATA_DIR:=$PWD/data}

export CMDPREFIX=${CMD_PREFIX:=echo "CMDR: "}
export CMDPOSTFIX=${CMD_POSTFIX:=}

if [[ ! -z $DEBUG ]]; then set -x; fi

echo "> Begin OpenWhisk Benchmark Trial"


function CMDR
{
  $CMDPREFIX $@ $CMDPOSTFIX
}

function SingleTrial
{
  local TPATH=$1
  FILES=$(/bin/bash -c "ls $TPATH | grep csv")
  
  echo "$FILES"

  TrialInit

  CMDR $OWDEPLOY Boot
  
  # IF argument is a directory
  for i in $FILES; 
  do 
    DoRun $TPATH$i; 
    CMDR $OWDEPLOY Reboot
  done

  CMDR $OWDEPLOY Shutdown
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
  local FN=$TRIALID
  if [[ -n $SEUSS ]]; then
    FN="$TRIALPATH/seuss_$NAME"
  else 
    FN="$TRIALPATH/linux_$NAME"
  fi
  
  CMDR touch $FN
  CMDR $OWWRAP --create --writeToFile --fileName $FN execFile $1
  
  #if successful
  if [[ -n $SEUSS ]]; then
    echo "> Finished SEUSS run: $NAME"
  else
    echo "> Finished LINUX run: $NAME"
  fi
  

  
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
