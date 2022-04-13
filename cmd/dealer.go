// Package cmd /*
package cmd

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
	"pbftnode/source/Launcher"
	"strconv"

	"github.com/spf13/cobra"
)

var dealerArg Launcher.DealerArg

// dealerCmd represents the dealer command
var dealerCmd = &cobra.Command{
	Use:   "dealer [IP:Port] [Port?]",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		dealerArg.BaseArg = baseArg.NewBaseArg(logLevel)
		var incomePort string
		dealerArg.Contact = args[0]
		if len(args) == 1 {
			var ok bool
			incomePort, ok = os.LookupEnv("IncomePort")
			if !ok {
				log.Fatal().Msg("The env variable IncomePort or the port argument is not set")
			}
		} else {
			incomePort = args[1]
		}
		_, err := strconv.Atoi(incomePort)
		if err != nil {
			log.Fatal().Msgf("The port need to be a number")
		}
		dealerArg.IncomePort = incomePort
		Launcher.Dealer(dealerArg)
		fmt.Println("closed")
	},
}

func init() {
	rootCmd.AddCommand(dealerCmd)
	dealerCmd.Flags().IntVarP(&dealerArg.NbOfNode, "NbNode", "N", 0, "Set the expected number of node")
	dealerCmd.Flags().BoolVar(&dealerArg.RandomDistrib, "RandomDistrib", false, "Use it to distribute the message to a random node")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// dealerCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// dealerCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
