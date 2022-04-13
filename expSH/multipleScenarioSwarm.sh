#!/bin/bash


dirName="exp_results_multiple_model0lag/"

#sleep infinity

shPath=$(dirname "$0")

mkdir $dirName

for idExp in $(seq 31 50); do
  "$shPath/scenarioDockerSwarm.sh" -c -p "$idExp" -s
  mv "exp/scenario$idExp" $dirName
done
