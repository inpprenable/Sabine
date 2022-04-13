#!/bin/bash

fileName=docker-compose-swarm.yml
NumberOfNode=8
CPUperNode="0.3"
MemLim="1024M"
MemReserv="384M"
lag=""
log="warn"
regularSave=10
txPoolBehavior="Drop"
controPeriod=12
refreshPeriod=5
SondeNumber=2
MetricTicker=5
typeLag="Fix"
scenario="$HOME/scenario"
modefDir="$HOME/model"
#ModelFile="3Dmap.csv"
FCB=""

lag=""
ModelType=""
FCB=""
modelFileOpt=""
save=""

nbArg=1

Help() {
  # Display Help
  echo "Create a yaml file use in the docker swarm."
  echo
  echo "Syntax: composeGen_swarm [options] NumberOfNode"
  echo "options:"
  printf "\t h  \t\t Print this Help. \n"
  printf "\t c float  \t Set the cpu per node. \n"
  printf "\t s \t\t Save in multiple file \n"
  printf "\t f \t\t Set the Feedback Control \n"
  printf "\t l int \t\t Set the lag \n"
  printf "\t g string \t Delay type (if lag) {NoDelay|Normal|Poisson|Fix} (default %s) \n" $typeLag
  printf "\t T string \t Set the FCB type {OneValidator|Hysteresis|ModelComparison} (default ModelComparison) \n"
  printf "\t m string \t Set the modelFile (default %s) \n" $ModelFile
  printf "\t F string \t Yaml filename (default %s) \n" $fileName
  printf "\t C string \t Chain repository for metrics, chain and logs (default %s) \n" $scenario
  echo
}

while getopts "hfg:l:c:sF:m:T:C: " opt; do
  case $opt in
  h)
    Help
    exit 0
    ;;
  f)
    FCB="--FCB"
    ;;
  g)
    typeLag="$OPTARG"
    ;;
  l)
    lag="--avgDelay $OPTARG"
    ;;
  c)
    CPUperNode=$OPTARG
    ;;
  s)
    save="--multiSaveFile"
    ;;
  F)
    fileName=$OPTARG
    ;;
  T)
    ModelType="--FCType $OPTARG"
    ;;
  m)
    ModelFile=$OPTARG
    if [ "$ModelFile" != "" ]; then
      modelFileOpt="--modelFile /home/model/$ModelFile"
    fi
    ;;
  C)
    scenario=$OPTARG
    ;;
  ' ') ;;
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

typeLag="--delayType $typeLag"
Flag="--RamOpt --PoA --txPoolBehavior $txPoolBehavior --debug $log --regularSave $regularSave $typeLag $lag $ModelType --ControlPeriod $controPeriod --RefreshingPeriod $refreshPeriod $FCB $modelFileOpt $save"

if [ $# == "$nbArg" ]; then
  NumberOfNode=$1
else
  echo "Error in the number of argument, need $nbArg, got $#" >&2
  Help
  exit 1
fi

node_order=($(uniq "$OAR_NODE_FILE" | grep -v "$HOSTNAME"))
nbG5Node=${#node_order[@]}

echo "version: '3.9'

services:
  bootstrap:
    container_name: bootstrap
    image: guilain/pbftnode:latest
    environment:
      BootPort: 4315
    deploy:
      placement:
        constraints:
          - 'node.role==manager'
    command: pbftnode bootstrap --debug trace 4315
" >$fileName

echo "

  dealer:
    container_name: dealer
    image: guilain/pbftnode:latest
    environment:
      IncomePort: 5000
    ports:
          - 5000:5000
    deploy:
      placement:
        constraints:
          - 'node.role==manager'
    command: pbftnode dealer bootstrap:4315 -N $NumberOfNode
" >>$fileName

echo "

  node0:
    container_name: node0
    image: guilain/pbftnode:latest
    depends_on:
      - bootstrap
    volumes:
      - $scenario/chain:/home/chain
      - $scenario/logs:/home/logs
      - $scenario/metric:/home/metric
      - $modefDir:/home/model
    deploy:
      restart_policy:
        condition: none
      placement:
        constraints:
          - 'node.hostname==${node_order[0]}'
      resources:
        limits:
          cpus: '$CPUperNode'
          memory: '$MemLim'
        reservations:
          cpus: '$CPUperNode'
          memory: '$MemReserv'
    stop_signal: SIGINT
    command: pbftnode node bootstrap:4315 0 --chainfile /home/chain/chain_0 --logFile /home/logs/log_0 -N $NumberOfNode $Flag" >>$fileName

for i in $(seq 1 $SondeNumber); do
  G5NodeId=$((i % nbG5Node))

  echo "

  node$i:
    container_name: node$i
    image: guilain/pbftnode:latest
    depends_on:
      - bootstrap
    volumes:
      - $scenario/chain:/home/chain
      - $scenario/logs:/home/logs
      - $scenario/metric:/home/metric
      - $modefDir:/home/model
    deploy:
      restart_policy:
        condition: none
      placement:
        constraints:
          - 'node.hostname==${node_order[G5NodeId]}'
      resources:
        limits:
          cpus: '$CPUperNode'
          memory: '$MemLim'
        reservations:
          cpus: '$CPUperNode'
          memory: '$MemReserv'
    stop_signal: SIGINT
    command: pbftnode node bootstrap:4315 $i --chainfile /home/chain/chain_$i --logFile /home/logs/log_$i -N $NumberOfNode $Flag --metricSaveFile /home/metric/metric_$i --metricTicker $MetricTicker" >>$fileName
done

MinimumNb=$((SondeNumber + 1))
NumberOfNodeLess=$((NumberOfNode - 1))
for i in $(seq $MinimumNb $NumberOfNodeLess); do
  G5NodeId=$((i % nbG5Node))
  echo "

  node$i:
    container_name: node$i
    image: guilain/pbftnode:latest
    depends_on:
      - bootstrap
    volumes:
      - $scenario/chain:/home/chain
      - $scenario/logs:/home/logs
      - $modefDir:/home/model
    deploy:
      restart_policy:
        condition: none
      placement:
        constraints:
          - 'node.hostname==${node_order[G5NodeId]}'
      resources:
        limits:
          cpus: '$CPUperNode'
          memory: '$MemLim'
        reservations:
          cpus: '$CPUperNode'
          memory: '$MemReserv'
    stop_signal: SIGINT
    command: pbftnode node bootstrap:4315 $i --chainfile /home/chain/chain_$i --logFile /home/logs/log_$i -N $NumberOfNode $Flag " >>$fileName
done


echo "

networks:
   default:
     driver: overlay
     ipam:
       driver: default
       config:
       - subnet:  10.6.0.0/16" >>$fileName