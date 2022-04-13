#!/bin/bash

#OAR -n docker_swarm_script
#OAR -l /nodes=5,walltime=13:00:00
#OAR -p cluster='grisou'
#OAR -t night
#OAR --stdout hello_world.out
#OAR --stderr hello_world.err

create_swarm() {
  export TAKTUK_CONNECTOR=oarsh
  taktuk -f <(uniq "$OAR_FILE_NODES") broadcast exec [ "/home/g5kcode/public/bin/g5k-setup-docker" ]
  taktuk -f <(uniq "$OAR_FILE_NODES") broadcast exec [ docker load -i ~/guilain.pbftnode.tar ]  #the docker image exported
  docker swarm init --advertise-addr br0
  join_token=$(docker swarm join-token worker | grep join)
  taktuk -f <(uniq "$OAR_FILE_NODES" | grep -v "$(hostname)") broadcast exec [ "$join_token" ]
}

dirName="results_alea/"

create_swarm

#sleep infinity

shPath=$(dirname "$0")

mkdir $dirName

for idExp in $(seq 10 20); do
  "$shPath/scenarioDockerSwarm_G5K.sh" -n 200 -c -p "$idExp" -s
  mv "scenario$idExp" $dirName
done
