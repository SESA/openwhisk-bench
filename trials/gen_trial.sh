######################################################################
#  Openwhisk Benchmakr Generate Trials
#  
#  USAGE: ./gen_trial.sh foo bar
#  Will populate trial scripts in the CWD:
#  	0,1,0,foo bar,1
#  	0,3,0,foo bar,1
#  	0,2,0,foo bar,1
#	...
######################################################################

export ICNT=${INVOCATION_CNT:=8192}
export POW2LIMIT=${POW2_LIMIT:=10}
export LABEL=${TRIAL_LABEL:=""}
export PARAM=${PARAMS:=}
if [ $# -eq 2 ]; then PARAM="${@}"; fi # Only support two params at the moment

TMPFILE=${TMP_FILE:=/tmp/gen_trial$RANDOM}
if [ -f $TMPFILE ]; then echo "Error: $TMPFILE exists"; exit 1; fi
touch $TMPFILE
for i in $(seq 0 $POW2LIMIT); do 
	USRS=$((1<<$i))
	PERUSR=$(($ICNT/$USRS))
	FILE=${LABEL}${ICNT}_${USRS}u.csv
	touch $FILE
	echo "Generating $FILE for $USRS users with $PERUSR invocations"
	for i in $(seq 1 $PERUSR); do 
		for j in $(seq 0 $((USRS-1))); do 
			if [ -z "$PARAM" ]; then
				echo "0,${j},0,1" >> $TMPFILE
			else
				echo "0,${j},0,${PARAM},1" >> $TMPFILE
			fi
		done
	done	
	cat $TMPFILE | shuf > $FILE
	true > $TMPFILE
done
rm $TMPFILE
