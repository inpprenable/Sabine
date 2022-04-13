#!/bin/bash

EXPDIR="$HOME/scenario"
bootIP="5000"

model="modele3DG5G.csv"
NbNode=64
lag=0
controle=0
multisave=1
delay=310
reduc=60

scenario="50:600 40:600 60:600 30:1200 0:600"
delayScenario="0:200 5:200 15:200 10:200 15:200 10:200 10:200 15:200 5:200 15:200 20:200 10:200 25:200 15:200 10:200 25:200"
durationDelayScenario=3600

Help() {
  # Display Help
  echo "Add description of the script functions here."
  echo
  echo "Syntax: scriptTemplate [-g|h|v|V]"
  echo "options:"
  printf "\t h \t\t Print this Help.\n"
  printf "\t n int \t\t Set the number of node (default %d).\n" $NbNode
  printf "\t v int \t\t Set the number of validator (default %d).\n" $((NbNode-reduc))
  printf "\t m int \t\t Set a fix emitted throughput during %d s.\n" $delay
  printf "\t d int \t\t Set a simulated delay (default %d).\n" $lag
  printf "\t s \t\t Set a random delay scenario during.\n"
  printf "\t n int \t\t Set the control loop.\n"
  printf "\t p string \t Set a salt used fo multiple experiment.\n"
  echo
}


fileName=docker-compose-test-swarm.yml
FCB=""
lagOpt=""
save=""
delayScenarioComm=""
modelParam=""

shPath=$(dirname "$0")

while getopts "hn:v:m:d:p:cs " opt; do
  case $opt in
  h)
    Help
    exit 0
    ;;
  n)
    NbNode=$OPTARG
    ;;
  v)
    reduc=$((NbNode-OPTARG))
    ;;
  m)
    scenario="$OPTARG:$delay"
    ;;
  ' ')
    ;;
  d)
    lag=$OPTARG
    ;;
  c)
    controle=1
    ;;
  p)
    salt="$OPTARG"
    EXPDIR="$EXPDIR""$salt"
    fileName="${fileName%.*}""$salt"".yaml"
    ;;
  s)
    delayScenario=$(python "$shPath/generateScenarioDelaySmart.py" -s "$scenario")
    ;;
  \?)
    echo "Invalid option: -$OPTARG l" >&2
    exit 1
    ;;
  :)
    echo "Option -$OPTARG requires an argument." >&2
    exit 1
    ;;
  esac
done
shift $((OPTIND - 1))
rm -r "$EXPDIR"
LOGDIR="$EXPDIR"/logs/
CHAINDIR="$EXPDIR"/chain/
METRICDIR="$EXPDIR"/metric/
mkdir -p "$LOGDIR"
mkdir -p "$CHAINDIR"
mkdir -p "$METRICDIR"
chmod a+rwx "$EXPDIR"

resume="NbNode=$NbNode
lag=$lag
scenario=$scenario
reduc=$reduc
model=$model
controle=$controle
delayScenario=$delayScenario
"

if [ $controle == 1 ]; then
  FCB="-f"
fi

if [ $multisave == 1 ]; then
  save="-s"
fi

if [ $lag -gt 0 ]; then
  lagOpt="-l $lag"
fi
if [ "$delayScenario" != "" ]; then
  delayScenarioComm="-d \"$delayScenario\""
fi
if [ "$model" != "" ]; then
  modelParam="-m $model"
fi

echo "$resume" >"$EXPDIR"/scenario.txt

param="$FCB $save $modelParam $lagOpt -C $EXPDIR -F $fileName $NbNode "
"$shPath/composeGen_swarmG5K.sh" $(echo "$param")

docker stack deploy --compose-file $fileName stackpbft

sleep=$(bc <<<"scale=2; $NbNode *0.5 +.1")
sleep $sleep

read -a gate <<<"$(docker network inspect docker_gwbridge | grep "Gateway")"
gateway=${gate[1]%\"}
bootAddr=${gateway#\"}

echo debut
docker run guilain/pbftnode pbftnode zombie throughput "$bootAddr:$bootIP" 2 $reduc $scenario -N $NbNode "$delayScenarioComm"
echo fin

docker stack rm stackpbft
sleep $sleep
echo fin de l\'expérience

if [ $multisave == 1 ]; then
  python "$shPath/smartCheckAndConcat.py" $CHAINDIR
fi

refChain=$(du -bs $CHAINDIR/* | sort -rn | head -n 1 | awk '{print $2;}')
refChain=$(basename "$refChain")
find "$CHAINDIR" ! -name "$refChain" -type f -exec rm -f {} \;
mv "$CHAINDIR/$refChain" "$CHAINDIR/chain"

echo fin des copies
