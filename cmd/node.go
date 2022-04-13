// Package cmd
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
package cmd

import (
	"github.com/spf13/cobra"
	"os"
	"pbftnode/source/Blockchain"
	"pbftnode/source/Launcher"
	"pbftnode/source/config"
)

var (
	nodeArg        Launcher.NodeArg
	controlType    string
	behaviorTxPool string
)

// nodeCmd represents the node command
var nodeCmd = &cobra.Command{
	Use:   "node [IP:Port] [ID]",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		nodeArg.BaseArg = baseArg.NewBaseArg(logLevel)
		nodeArg.BootAddr = args[0]
		nodeArg.NodeId = args[1]
		if nodeArg.ListeningPort == "" {
			bootPort, ok := os.LookupEnv("ListenPort")
			if ok {
				nodeArg.ListeningPort = bootPort
			}
		}
		nodeArg.Param.ControlType = Blockchain.ControlTypeStr(controlType)
		nodeArg.Param.Behavior = Blockchain.StrToBehavior(behaviorTxPool)

		Launcher.Node(nodeArg)
	},
}

func init() {
	rootCmd.AddCommand(nodeCmd)
	//nodeCmd.Flags().BoolP("Broadcast", "b", true, "Use it if you want to broadcast every received message")
	nodeCmd.Flags().BoolVarP(&nodeArg.Param.Broadcast, "Broadcasting", "b", false, "Use it if you want to broadcast every received message")
	nodeCmd.Flags().IntVarP(&nodeArg.NodeNumber, "NodeNumber", "N", config.NumberOfNodes, "The number of Nodes in the networks")
	nodeCmd.Flags().BoolVar(&nodeArg.Param.PoANV, "PoA", false, "Use it if you want the non Validator listen from the proposer")
	nodeCmd.Flags().StringVar(&nodeArg.SaveFile, "chainfile", "", "The file used to store the blockchain")
	nodeCmd.Flags().IntVar(&nodeArg.AvgDelay, "avgDelay", 0, "Additional average delay of transition (ms)")
	nodeCmd.Flags().StringVar(&nodeArg.DelayType, "delayType", "NoDelay", "Delay type (if avgDelay>0) {NoDelay|Normal|Poisson|Fix}")
	nodeCmd.Flags().IntVar(&nodeArg.StdDelay, "stdDelay", 10, "The standard deviation for the additional delay, if normal law")
	nodeCmd.Flags().StringVar(&nodeArg.HttpChain, "httpChain", "", "The HTTP port to export the chain")
	nodeCmd.Flags().StringVar(&nodeArg.HttpMetric, "httpMetric", "", "The HTTP port to view current data")
	nodeCmd.Flags().BoolVar(&nodeArg.Param.RamOpt, "RamOpt", false, "Use it to remove old message")
	nodeCmd.Flags().BoolVar(&nodeArg.PPRof, "pprof", false, "Use it to star a http server for on port 6060 for pprof")
	nodeCmd.Flags().BoolVar(&nodeArg.Control, "FCB", false, "Use it to set FeedBack Control")
	nodeCmd.Flags().StringVar(&controlType, "FCType", "ModelComparison", "Set the FCB type {OneValidator|Hysteresis|ModelComparison}")
	nodeCmd.Flags().StringVar(&nodeArg.Param.ModelFile, "modelFile", "", "Set the csv file which contains the model ")
	nodeCmd.Flags().StringVarP(&nodeArg.ListeningPort, "listeningPort", "p", "", "Listening port for the node")
	nodeCmd.Flags().IntVar(&nodeArg.Sleep, "sleep", 0, "Sleep n milliseconds before the connection")
	nodeCmd.Flags().StringVar(&nodeArg.Param.MetricSaveFile, "metricSaveFile", "", "Use it to save measured metric in the specified file")
	nodeCmd.Flags().IntVar(&nodeArg.Param.TickerSave, "metricTicker", 0, "Make a regular save of metrics every x minutes")
	nodeCmd.Flags().IntVar(&nodeArg.RegularSave, "regularSave", 0, "Make a regular save of the chain every x minutes")
	nodeCmd.Flags().StringVar(&behaviorTxPool, "txPoolBehavior", "Nothing", "Change the behavior of the transaction pool when threshold is reach {Nothing|Ignore|Drop}")
	nodeCmd.Flags().IntVar(&nodeArg.Param.RefreshingPeriod, "RefreshingPeriod", 1, "Change the refreshing Period of metrics computation (in seconds)")
	nodeCmd.Flags().IntVar(&nodeArg.Param.ControlPeriod, "ControlPeriod", 10, "Change the control period of FCB (x * RefreshingPeriod)")
	nodeCmd.Flags().BoolVar(&nodeArg.MultiSaveFile, "multiSaveFile", false, "Use it to save on multiple files instead of one")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// nodeCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// nodeCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
