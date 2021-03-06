// VulcanizeDB
// Copyright © 2019 Vulcanize

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"fmt"
	"time"

	st "github.com/makerdao/vulcanizedb/libraries/shared/transformer"
	"github.com/makerdao/vulcanizedb/pkg/config"
	ft "github.com/makerdao/vulcanizedb/pkg/contract_watcher/full/transformer"
	ht "github.com/makerdao/vulcanizedb/pkg/contract_watcher/header/transformer"
	"github.com/makerdao/vulcanizedb/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// contractWatcherCmd represents the contractWatcher command
var contractWatcherCmd = &cobra.Command{
	Use:   "contractWatcher",
	Short: "Watches events at the provided contract address using fully synced vDB",
	Long: `Uses input contract address and event filters to watch events

Expects an ethereum node to be running
Expects an archival node synced into vulcanizeDB
Requires a .toml config file:

  [database]
    name     = "vulcanize_public"
    hostname = "localhost"
    port     = 5432

  [client]
    ipcPath  = "/Users/user/Library/Ethereum/geth.ipc"

  [contract]
    network  = ""
    addresses  = [
        "contractAddress1",
        "contractAddress2"
    ]
    [contract.contractAddress1]
        abi    = 'ABI for contract 1'
        startingBlock = 982463
    [contract.contractAddress2]
        abi    = 'ABI for contract 2'
        events = [
            "event1",
            "event2"
        ]
		eventArgs = [
			"arg1",
			"arg2"
		]
        methods = [
            "method1",
			"method2"
        ]
		methodArgs = [
			"arg1",
			"arg2"
		]
        startingBlock = 4448566
        piping = true
`,
	Run: func(cmd *cobra.Command, args []string) {
		SubCommand = cmd.CalledAs()
		LogWithCommand = *logrus.WithField("SubCommand", SubCommand)
		contractWatcher()
	},
}

var (
	mode string
)

func contractWatcher() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	blockChain := getBlockChain()
	db := utils.LoadPostgres(databaseConfig, blockChain.Node())

	var t st.ContractTransformer
	con := config.ContractConfig{}
	con.PrepConfig()
	switch mode {
	case "header":
		t = ht.NewTransformer(con, blockChain, &db)
	case "full":
		t = ft.NewTransformer(con, blockChain, &db)
	default:
		LogWithCommand.Fatal("Invalid mode")
	}

	err := t.Init()
	if err != nil {
		LogWithCommand.Fatal(fmt.Sprintf("Failed to initialize transformer, err: %v ", err))
	}

	for range ticker.C {
		err = t.Execute()
		if err != nil {
			LogWithCommand.Error("Execution error for transformer: ", t.GetConfig().Name, err)
		}
	}
}

func init() {
	rootCmd.AddCommand(contractWatcherCmd)
	contractWatcherCmd.Flags().StringVarP(&mode, "mode", "o", "header", "'header' or 'full' mode to work with either header synced or fully synced vDB (default is header)")
}
