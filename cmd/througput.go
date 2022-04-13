package cmd

/*
Copyright © 2021 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"pbftnode/source/Launcher"
	"strconv"
	"strings"
)

var (
	throughputArg Launcher.ZombieTxArg
)

// ThroughputCmd represents the client command
var ThroughputCmd = &cobra.Command{
	Use:   "throughput [IP:Port] [NodeId] [Reducing Validator] [nb of Tx per second:exp duration]",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.MinimumNArgs(4),
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		zombieArgCreate(args)
		throughputArg.ZombieArg = zombieArg
		scenario := args[3:]
		for _, scenariiText := range scenario {
			if scenariiText != "" {
				var scenarii Launcher.Scenarii
				seq := strings.Split(scenariiText, ":")
				scenarii.NbTxPS, err = strconv.Atoi(seq[0])
				if err != nil {
					panic("[nb of Tx] must be an int")
				}
				scenarii.Duration, err = strconv.Atoi(seq[1])
				if err != nil {
					log.Panic().Msgf("[exp duration] must be an int, get %T", seq[1])
				}
				throughputArg.Scenario = append(throughputArg.Scenario, scenarii)
			}
		}
		Launcher.ZombieTx(throughputArg)
	},
}

func init() {
	ZombieCmd.AddCommand(ThroughputCmd)
	ThroughputCmd.Flags().BoolVar(&throughputArg.Multi, "multi", false, "Use it to distribute transactions to all nodes")
	ThroughputCmd.Flags().IntVarP(&throughputArg.NbOfNode, "NbNode", "N", 0, "Wait until N nodes are connected to the bootstrap server")
	ThroughputCmd.Flags().StringVarP(&throughputArg.DelayScenarioStr, "delayScenario", "d", "", "Specify the scenario for the evolution of the delai")

}
