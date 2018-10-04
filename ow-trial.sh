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
    DoRun $FPATH$i; 
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
    FN="$TRIALID/seuss_$NAME"
  else 
    FN="$TRIALID/linux_$NAME"
  fi
  
  CMDR $OWWRAP --create --writeToFile --fileName $FN execFile $1
  
  #if successful
  if [[ -n $SEUSS ]]; then
    echo "> Finished SEUSS run: $NAME"
  else
    echo "> Finished LINUX run: $NAME"
  fi
  

  
}






#function OldRun 
#{
#  local test_file=$1
#  if [[ -z $test_file ]];
#  then
#    echo "Error: Cannot Run Tests; Test File Path Needed"
#    return
#  fi
#
#  if [[ "$2" = "--stats" ]];
#  then
#    if [[ -f temp.csv ]];
#    then
#      rm temp.csv
#    fi
#    if [[ -f stats.csv ]];
#    then
#      rm stats.csv
#    fi
#
#    go run *.go --create execFile $test_file >> temp.csv
#    local total_time="$(cat temp.csv | awk '/Total_Job_Time/ {print}')"
#
#    cat temp.csv | tail -n+9 | head -n -3 >> stats.csv
#    local numInvocations="$(cat stats.csv | wc -l)"
#    local total_wait="$(cat stats.csv | awk -F "\"*,\"*" '{print $6}' | paste -sd+ | bc)"
#    local total_init="$(cat stats.csv | awk -F "\"*,\"*" '{print $7}' | paste -sd+ | bc)"
#    local total_duration="$(cat stats.csv | awk -F "\"*,\"*" '{print $8}' | paste -sd+ | bc)"
#    local total_rtt="$(cat stats.csv | awk -F "\"*,\"*" '{print $9}' | paste -sd+ | bc)"
#    
#    # averages
#    local avg_wait="$((total_wait / numInvocations))"
#    local avg_init="$((total_init / numInvocations))"
#    local avg_duration="$((total_duration / numInvocations))"
#    local avg_rtt="$((total_rtt / numInvocations))"
#
#    # min, max
#    local min_wait="$(cat stats.csv | awk -F "\"*,\"*" '{print $6}' | sort -n | head -1)"
#    local max_wait="$(cat stats.csv | awk -F "\"*,\"*" '{print $6}' | sort -n | tail -1)"
#    local min_init="$(cat stats.csv | awk -F "\"*,\"*" '{print $7}' | sort -n | head -1)"
#    local max_init="$(cat stats.csv | awk -F "\"*,\"*" '{print $7}' | sort -n | tail -1)"
#    local min_duration="$(cat stats.csv | awk -F "\"*,\"*" '{print $8}' | sort -n | head -1)"
#    local max_duration="$(cat stats.csv | awk -F "\"*,\"*" '{print $8}' | sort -n | tail -1)"
#    local min_rtt="$(cat stats.csv | awk -F "\"*,\"*" '{print $9}' | sort -n | head -1)"
#    local max_rtt="$(cat stats.csv | awk -F "\"*,\"*" '{print $9}' | sort -n | tail -1)"
#
#    # stddev
#    local wait_stdev="$(cat stats.csv | awk -F "\"*,\"*" '{print $6}' | awk '{sum+=$1; sumsq+=$1*$1}END{print int(sqrt(sumsq/NR - (sum/NR)**2))}')"
#    local init_stdev="$(cat stats.csv | awk -F "\"*,\"*" '{print $7}' | awk '{sum+=$1; sumsq+=$1*$1}END{print int(sqrt(sumsq/NR - (sum/NR)**2))}')"
#    local duration_stdev="$(cat stats.csv | awk -F "\"*,\"*" '{print $8}' | awk '{sum+=$1; sumsq+=$1*$1}END{print int(sqrt(sumsq/NR - (sum/NR)**2))}')"
#    local rtt_stdev="$(cat stats.csv | awk -F "\"*,\"*" '{print $9}' | awk '{sum+=$1; sumsq+=$1*$1}END{print int(sqrt(sumsq/NR - (sum/NR)**2))}')"
#
#    echo "See Full Output in Temporary File: stats.csv"
#    echo ""
#    echo "Total_Wait_Time: $total_wait"
#    echo "Total_Init_Time: $total_init"
#    echo "Total_Duration_Time: $total_duration"
#    echo ""
#    echo "Average_Wait_Time: $avg_wait"
#    echo "Min_Wait_Time: $min_wait"
#    echo "Max_Wait_Time: $max_wait"
#    echo "Stdev_Wait: $wait_stdev"
#    echo ""
#    echo "Average_Init_Time: $avg_init"
#    echo "Min_Init_Time: $min_init"
#    echo "Max_Init_Time: $max_init"
#    echo "Stdev_Init: $init_stdev"
#    echo ""
#    echo "Average_Duration_Time: $avg_duration"
#    echo "Min_Duration_Time: $min_duration"
#    echo "Max_Duration_Time: $max_duration"
#    echo "Stdev_Duration: $duration_stdev"
#    echo ""
#    echo "Average_RTT_Time: $avg_rtt"
#    echo "Min_RTT_Time: $min_rtt"
#    echo "Max_RTT_Time: $max_rtt"
#    echo "Stdev_RTT: $rtt_stdev"
#    echo ""
#    echo "$total_time"
#
#  else
#    go run *.go --create execFile $test_file
#  fi
#}
#
#######################################################################
#
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
