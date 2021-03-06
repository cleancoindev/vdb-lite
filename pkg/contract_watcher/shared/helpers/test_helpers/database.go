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

package test_helpers

import (
	"math/rand"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	. "github.com/onsi/gomega"

	"github.com/makerdao/vulcanizedb/pkg/config"
	"github.com/makerdao/vulcanizedb/pkg/contract_watcher/shared/constants"
	"github.com/makerdao/vulcanizedb/pkg/contract_watcher/shared/contract"
	"github.com/makerdao/vulcanizedb/pkg/contract_watcher/shared/helpers/test_helpers/mocks"
	"github.com/makerdao/vulcanizedb/pkg/core"
	"github.com/makerdao/vulcanizedb/pkg/datastore/postgres"
	"github.com/makerdao/vulcanizedb/pkg/datastore/postgres/repositories"
	"github.com/makerdao/vulcanizedb/pkg/eth"
	"github.com/makerdao/vulcanizedb/pkg/eth/client"
	rpc2 "github.com/makerdao/vulcanizedb/pkg/eth/converters/rpc"
	"github.com/makerdao/vulcanizedb/pkg/eth/node"
	"github.com/makerdao/vulcanizedb/test_config"
)

type TransferLog struct {
	Id             int64  `db:"id"`
	VulcanizeLogId int64  `db:"vulcanize_log_id"`
	TokenName      string `db:"token_name"`
	Block          int64  `db:"block"`
	Tx             string `db:"tx"`
	From           string `db:"from_"`
	To             string `db:"to_"`
	Value          string `db:"value_"`
}

type NewOwnerLog struct {
	Id             int64  `db:"id"`
	VulcanizeLogId int64  `db:"vulcanize_log_id"`
	TokenName      string `db:"token_name"`
	Block          int64  `db:"block"`
	Tx             string `db:"tx"`
	Node           string `db:"node_"`
	Label          string `db:"label_"`
	Owner          string `db:"owner_"`
}

type HeaderSyncTransferLog struct {
	Id        int64  `db:"id"`
	HeaderID  int64  `db:"header_id"`
	TokenName string `db:"token_name"`
	LogIndex  int64  `db:"log_idx"`
	TxIndex   int64  `db:"tx_idx"`
	From      string `db:"from_"`
	To        string `db:"to_"`
	Value     string `db:"value_"`
	RawLog    []byte `db:"raw_log"`
}

type HeaderSyncNewOwnerLog struct {
	Id        int64  `db:"id"`
	HeaderID  int64  `db:"header_id"`
	TokenName string `db:"token_name"`
	LogIndex  int64  `db:"log_idx"`
	TxIndex   int64  `db:"tx_idx"`
	Node      string `db:"node_"`
	Label     string `db:"label_"`
	Owner     string `db:"owner_"`
	RawLog    []byte `db:"raw_log"`
}

type BalanceOf struct {
	Id        int64  `db:"id"`
	TokenName string `db:"token_name"`
	Block     int64  `db:"block"`
	Address   string `db:"who_"`
	Balance   string `db:"returned"`
}

type Resolver struct {
	Id        int64  `db:"id"`
	TokenName string `db:"token_name"`
	Block     int64  `db:"block"`
	Node      string `db:"node_"`
	Address   string `db:"returned"`
}

type Owner struct {
	Id        int64  `db:"id"`
	TokenName string `db:"token_name"`
	Block     int64  `db:"block"`
	Node      string `db:"node_"`
	Address   string `db:"returned"`
}

func SetupDBandBC() (*postgres.DB, core.BlockChain) {
	con := test_config.TestClient
	testIPC := con.IPCPath
	rawRpcClient, err := rpc.Dial(testIPC)
	Expect(err).NotTo(HaveOccurred())
	rpcClient := client.NewRpcClient(rawRpcClient, testIPC)
	ethClient := ethclient.NewClient(rawRpcClient)
	blockChainClient := client.NewEthClient(ethClient)
	madeNode := node.MakeNode(rpcClient)
	transactionConverter := rpc2.NewRpcTransactionConverter(ethClient)
	blockChain := eth.NewBlockChain(blockChainClient, rpcClient, madeNode, transactionConverter)

	db, err := postgres.NewDB(config.Database{
		Hostname: "localhost",
		Name:     "vulcanize_testing",
		Port:     5432,
	}, blockChain.Node())
	Expect(err).NotTo(HaveOccurred())

	return db, blockChain
}

func SetupTusdRepo(vulcanizeLogId *int64, wantedEvents, wantedMethods []string) (*postgres.DB, *contract.Contract) {
	db, err := postgres.NewDB(config.Database{
		Hostname: "localhost",
		Name:     "vulcanize_testing",
		Port:     5432,
	}, core.Node{})
	Expect(err).NotTo(HaveOccurred())

	receiptRepository := repositories.FullSyncReceiptRepository{DB: db}
	logRepository := repositories.FullSyncLogRepository{DB: db}
	blockRepository := *repositories.NewBlockRepository(db)

	blockNumber := rand.Int63()
	blockId := CreateBlock(blockNumber, blockRepository)

	receipts := []core.Receipt{{Logs: []core.FullSyncLog{{}}}}

	err = receiptRepository.CreateReceiptsAndLogs(blockId, receipts)
	Expect(err).ToNot(HaveOccurred())

	err = logRepository.Get(vulcanizeLogId, `SELECT id FROM full_sync_logs`)
	Expect(err).ToNot(HaveOccurred())

	info := SetupTusdContract(wantedEvents, wantedMethods)

	return db, info
}

func SetupTusdContract(wantedEvents, wantedMethods []string) *contract.Contract {
	p := mocks.NewParser(constants.TusdAbiString)
	err := p.Parse()
	Expect(err).ToNot(HaveOccurred())

	return contract.Contract{
		Name:          "TrueUSD",
		Address:       constants.TusdContractAddress,
		Abi:           p.Abi(),
		ParsedAbi:     p.ParsedAbi(),
		StartingBlock: 6194634,
		Events:        p.GetEvents(wantedEvents),
		Methods:       p.GetSelectMethods(wantedMethods),
		MethodArgs:    map[string]bool{},
		FilterArgs:    map[string]bool{},
	}.Init()
}

func SetupENSRepo(vulcanizeLogId *int64, wantedEvents, wantedMethods []string) (*postgres.DB, *contract.Contract) {
	db, err := postgres.NewDB(config.Database{
		Hostname: "localhost",
		Name:     "vulcanize_testing",
		Port:     5432,
	}, core.Node{})
	Expect(err).NotTo(HaveOccurred())

	receiptRepository := repositories.FullSyncReceiptRepository{DB: db}
	logRepository := repositories.FullSyncLogRepository{DB: db}
	blockRepository := *repositories.NewBlockRepository(db)

	blockNumber := rand.Int63()
	blockId := CreateBlock(blockNumber, blockRepository)

	receipts := []core.Receipt{{Logs: []core.FullSyncLog{{}}}}

	err = receiptRepository.CreateReceiptsAndLogs(blockId, receipts)
	Expect(err).ToNot(HaveOccurred())

	err = logRepository.Get(vulcanizeLogId, `SELECT id FROM full_sync_logs`)
	Expect(err).ToNot(HaveOccurred())

	info := SetupENSContract(wantedEvents, wantedMethods)

	return db, info
}

func SetupENSContract(wantedEvents, wantedMethods []string) *contract.Contract {
	p := mocks.NewParser(constants.ENSAbiString)
	err := p.Parse()
	Expect(err).ToNot(HaveOccurred())

	return contract.Contract{
		Name:          "ENS-Registry",
		Address:       constants.EnsContractAddress,
		Abi:           p.Abi(),
		ParsedAbi:     p.ParsedAbi(),
		StartingBlock: 6194634,
		Events:        p.GetEvents(wantedEvents),
		Methods:       p.GetSelectMethods(wantedMethods),
		MethodArgs:    map[string]bool{},
		FilterArgs:    map[string]bool{},
	}.Init()
}

func SetupMarketPlaceContract(wantedEvents, wantedMethods []string) *contract.Contract {
	p := mocks.NewParser(constants.MarketPlaceAbiString)
	err := p.Parse()
	Expect(err).NotTo(HaveOccurred())

	return contract.Contract{
		Name:          "Marketplace",
		Address:       constants.MarketPlaceContractAddress,
		StartingBlock: 6496012,
		Abi:           p.Abi(),
		ParsedAbi:     p.ParsedAbi(),
		Events:        p.GetEvents(wantedEvents),
		Methods:       p.GetSelectMethods(wantedMethods),
		FilterArgs:    map[string]bool{},
		MethodArgs:    map[string]bool{},
	}.Init()
}

func SetupMolochContract(wantedEvents, wantedMethods []string) *contract.Contract {
	p := mocks.NewParser(constants.MolochAbiString)
	err := p.Parse()
	Expect(err).NotTo(HaveOccurred())

	return contract.Contract{
		Name:          "Moloch",
		Address:       constants.MolochContractAddress,
		StartingBlock: 7218566,
		Abi:           p.Abi(),
		ParsedAbi:     p.ParsedAbi(),
		Events:        p.GetEvents(wantedEvents),
		Methods:       p.GetSelectMethods(wantedMethods),
		FilterArgs:    map[string]bool{},
		MethodArgs:    map[string]bool{},
	}.Init()
}

// TODO: tear down/setup DB from migrations so this doesn't alter the schema between tests
func TearDown(db *postgres.DB) {
	tx, err := db.Beginx()
	Expect(err).NotTo(HaveOccurred())

	_, err = tx.Exec(`DELETE FROM addresses`)
	Expect(err).NotTo(HaveOccurred())

	_, err = tx.Exec(`DELETE FROM blocks`)
	Expect(err).NotTo(HaveOccurred())

	_, err = tx.Exec(`DELETE FROM headers`)
	Expect(err).NotTo(HaveOccurred())

	_, err = tx.Exec(`DELETE FROM full_sync_logs`)
	Expect(err).NotTo(HaveOccurred())

	_, err = tx.Exec(`DELETE FROM log_filters`)
	Expect(err).NotTo(HaveOccurred())

	_, err = tx.Exec(`DELETE FROM full_sync_transactions`)
	Expect(err).NotTo(HaveOccurred())

	_, err = tx.Exec("DELETE FROM header_sync_transactions")
	Expect(err).NotTo(HaveOccurred())

	_, err = tx.Exec(`DELETE FROM full_sync_receipts`)
	Expect(err).NotTo(HaveOccurred())

	_, err = tx.Exec(`DELETE FROM header_sync_receipts`)
	Expect(err).NotTo(HaveOccurred())

	_, err = tx.Exec(`DROP TABLE checked_headers`)
	Expect(err).NotTo(HaveOccurred())

	_, err = tx.Exec(`CREATE TABLE checked_headers (
    	id SERIAL PRIMARY KEY,
    	header_id INTEGER UNIQUE NOT NULL REFERENCES headers (id) ON DELETE CASCADE);`)
	Expect(err).NotTo(HaveOccurred())

	_, err = tx.Exec(`DROP SCHEMA IF EXISTS full_0x8dd5fbce2f6a956c3022ba3663759011dd51e73e CASCADE`)
	Expect(err).NotTo(HaveOccurred())

	_, err = tx.Exec(`DROP SCHEMA IF EXISTS header_0x8dd5fbce2f6a956c3022ba3663759011dd51e73e CASCADE`)
	Expect(err).NotTo(HaveOccurred())

	_, err = tx.Exec(`DROP SCHEMA IF EXISTS full_0x314159265dd8dbb310642f98f50c066173c1259b CASCADE`)
	Expect(err).NotTo(HaveOccurred())

	_, err = tx.Exec(`DROP SCHEMA IF EXISTS header_0x314159265dd8dbb310642f98f50c066173c1259b CASCADE`)
	Expect(err).NotTo(HaveOccurred())

	err = tx.Commit()
	Expect(err).NotTo(HaveOccurred())

	_, err = db.Exec(`VACUUM checked_headers`)
	Expect(err).NotTo(HaveOccurred())
}

func CreateBlock(blockNumber int64, repository repositories.BlockRepository) (blockId int64) {
	blockId, err := repository.CreateOrUpdateBlock(core.Block{Number: blockNumber})
	Expect(err).NotTo(HaveOccurred())

	return blockId
}
