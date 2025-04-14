#!/bin/bash

fileName=docker-compose-test-swarm.yml
NumberOfNode=8
CPUperNode="0.5"
MemLim="1024M"
MemReserv="384M"
lag=""
log="error"
nbPi=10
regularSave=55
txPoolBehavior="Drop"
controPeriod=12
refreshPeriod=5
SondeNumber=2
MetricTicker=5
typeLag="Fix"
ModelFile="/home/model/model.csv"
FCB=""

nbArg=4

if [ $# == "$nbArg" ]; then
  fileName=$1
  NumberOfNode=$2
  nbPi=$3
  lag=""
  if [ $4 == "FCB" ]; then
    FCB="--FCB"
  fi
elif [ $# == 5 ]; then
  fileName=$1
  NumberOfNode=$2
  nbPi=$3
  if [ $4 == "FCB" ]; then
    FCB="--FCB"
  fi
  lag="--delayType $typeLag --avgDelay $5"
else
  echo "composeGen_swarm.sh [fileName] [NumberOfNode] [nbPi] [FCB] [lag ?]"
fi

unusedFlag="--multiSaveFile"
Flag="--RamOpt --PoA --txPoolBehavior $txPoolBehavior --debug $log --regularSave $regularSave $lag --modelFile $ModelFile --FCType ModelComparison --ControlPeriod $controPeriod --RefreshingPeriod $refreshPeriod $FCB"

echo "version: '3.9'

services:
  bootstrap:
    container_name: bootstrap
    image: guilain/pbftnode
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
    image: guilain/pbftnode
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
    image: guilain/pbftnode_armv8
    depends_on:
      - bootstrap
    ports:
      - 6060:6060
    volumes:
      - /exp/scenario/chain:/home/chain
      - /exp/scenario/logs:/home/logs
      - /home/ubuntu/model:/home/model
    deploy:
      restart_policy:
        condition: none
      placement:
        constraints:
          - 'node.hostname==pi1'
      resources:
        limits:
          cpus: '$CPUperNode'
          memory: '$MemLim'
        reservations:
          cpus: '$CPUperNode'
          memory: '$MemReserv'
    stop_signal: SIGINT
    command: pbftnode node bootstrap:4315 0 --chainfile /home/chain/chain_0 --logFile /home/logs/log_0 -N $NumberOfNode $Flag" >>$fileName
#--metricSaveFile /home/metric/metric_0

for i in $(seq 1 $SondeNumber); do
  PiId=$((i % nbPi + 1))

  echo "

  node$i:
    container_name: node$i
    image: guilain/pbftnode_armv8
    depends_on:
      - bootstrap
    volumes:
      - /exp/scenario/chain:/home/chain
      - /exp/scenario/logs:/home/logs
      - /exp/scenario/metric:/home/metric
      - /home/ubuntu/model:/home/model
    deploy:
      restart_policy:
        condition: none
      placement:
        constraints:
          - 'node.hostname==pi$PiId'
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
  PiId=$((i % nbPi + 1))
  echo "

  node$i:
    container_name: node$i
    image: guilain/pbftnode_armv8
    depends_on:
      - bootstrap
    volumes:
      - /exp/scenario/chain:/home/chain
      - /exp/scenario/logs:/home/logs
      - /home/ubuntu/model:/home/model
    deploy:
      restart_policy:
        condition: none
      placement:
        constraints:
          - 'node.hostname==pi$PiId'
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
