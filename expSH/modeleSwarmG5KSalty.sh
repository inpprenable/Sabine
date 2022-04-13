#!/bin/bash

#OAR -n docker_swarm_script
#OAR -l /nodes=5,walltime=13:00:00
#OAR -p cluster='grisou'
#OAR -t night
#OAR --stdout hello_world1.out
#OAR --stderr hello_world1.err

create_swarm() {
  export TAKTUK_CONNECTOR=oarsh
  taktuk -f <(uniq "$OAR_FILE_NODES") broadcast exec [ "/home/g5kcode/public/bin/g5k-setup-docker" ]
  taktuk -f <(uniq "$OAR_FILE_NODES") broadcast exec [ docker load -i ~/guilain.pbftnode.tar ]
  docker swarm init  --advertise-addr br0
  join_token=$(docker swarm join-token worker | grep join)
  taktuk -f <(uniq "$OAR_FILE_NODES" | grep -v "$(hostname)") broadcast exec [ "$join_token" ]
}

function fileExist() {
  file=$1
  salt=0
  fileSalty="$file"
  while [ -f "$fileSalty" ]; do
    salt=$((salt + 1))
    fileSalty="$file"_"$salt"
  done
  echo "$fileSalty"
}

create_swarm

shPath=$(dirname "$0")
resultsDir="results_modele"

listValid="4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 21 23 25 27 29 31 33 35 37 39 41 43 45 50 55 60 65 70 75 80 85 90 95 100 110 120 130 140 150 160 170 180 190 200"
listDelay="0"
Salt=1
nb_node=200

mkdir -p "$resultsDir"
for delay in $listDelay; do
  mkdir -p "$resultsDir/lag_$delay"
  for nb_val in $listValid; do
    debit=$(python "$shPath"/calcDebitG5K.py "$nb_val")
    "$shPath/scenarioDockerSwarm_G5K.sh" -n "$nb_node" -v "$nb_val" -d "$delay" -m "$debit" -p "_$Salt"
    chainName=$(fileExist "$resultsDir/lag_$delay/chain_$nb_val""_""$nb_node")
    mv "$HOME/scenario_$Salt/chain/chain" "$chainName"
  done
done
