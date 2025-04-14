#!/bin/bash

EXPDIR="./exp/scenario"
bootIP="5000"
bootAddr="localhost"
NbNode=50
lag=0
controle=1


scenario="50:600 40:600 60:600 30:1000 0:1 30:1000 0:1 30:1000 0:600"

reduc=0
log="error"
nbPI=10

fileName=docker-compose-test-swarm.yml
FCB=""

LOGDIR="$EXPDIR"/logs/
CHAINDIR="$EXPDIR"/chain/
METRICDIR="$EXPDIR"/metric/
mkdir -p "$LOGDIR"
mkdir -p "$CHAINDIR"
mkdir -p "$METRICDIR"

resume="NbNode=$NbNode
lag=$lag
scenario=$scenario
reduc=$reduc
log=$log
nbPI=$nbPI
controle=$controle
"

if [ $controle == 1 ];then
  FCB="FCB"
else
  FCB="noFCB"
fi

echo "$resume" > "$EXPDIR"/scenario.txt

shPath=$(dirname "$0")
"$shPath/composeGen_swarm.sh" $fileName $NbNode $nbPI $FCB $lag


parallel-ssh -h /etc/ssh/pssh_host/pssh -I < expSH/restoreExp.sh
docker stack deploy --compose-file $fileName stackpbft

sleep=$(bc <<< "scale=2; $NbNode *0.5 +.1")
sleep $sleep

echo debut
./pbftnode zombie throughput "$bootAddr:$bootIP" 2 $reduc $scenario --debug error -N $NbNode
echo fin

docker stack rm stackpbft
sleep $sleep
echo fin de l\'expérience

listPID=""
for i in $(seq 1 $nbPI); do
  scp -Cr "ubuntu@3.14.15.$i:/exp/scenario/*" $EXPDIR &
  listPID="$listPID $!"
done

for PID in $listPID; do
  wait "$PID"
done


echo fin des copies
