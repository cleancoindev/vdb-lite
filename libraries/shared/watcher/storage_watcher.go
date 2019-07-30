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

package watcher

import (
	"fmt"
	"reflect"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/sirupsen/logrus"

	"github.com/vulcanize/vulcanizedb/libraries/shared/fetcher"
	"github.com/vulcanize/vulcanizedb/libraries/shared/storage"
	"github.com/vulcanize/vulcanizedb/libraries/shared/storage/utils"
	"github.com/vulcanize/vulcanizedb/libraries/shared/transformer"
	"github.com/vulcanize/vulcanizedb/pkg/datastore/postgres"
)

type IStorageWatcher interface {
	AddTransformers(initializers []transformer.StorageTransformerInitializer)
	Execute(diffsChan chan utils.StorageDiff, errsChan chan error, queueRecheckInterval time.Duration)
}

type StorageWatcher struct {
	db                        *postgres.DB
	StorageFetcher            fetcher.IStorageFetcher
	Queue                     storage.IStorageQueue
	Transformers              map[common.Address]transformer.StorageTransformer
	KeccakAddressTransformers map[common.Address]transformer.StorageTransformer // keccak hash of an address => transformer
	transformerGetter         func(common.Address) (transformer.StorageTransformer, bool)
}

func (storageWatcher StorageWatcher) AddTransformers(initializers []transformer.StorageTransformerInitializer) {
	for _, initializer := range initializers {
		storageTransformer := initializer(storageWatcher.db)
		storageWatcher.Transformers[storageTransformer.ContractAddress()] = storageTransformer
	}
}

func (storageWatcher StorageWatcher) Execute(diffsChan chan utils.StorageDiff, errsChan chan error, queueRecheckInterval time.Duration) {
	ticker := time.NewTicker(queueRecheckInterval)
	go storageWatcher.StorageFetcher.FetchStorageDiffs(diffsChan, errsChan)
	for {
		select {
		case fetchErr := <-errsChan:
			logrus.Warn(fmt.Sprintf("error fetching storage diffs: %s", fetchErr))
		case diff := <-diffsChan:
			storageWatcher.processRow(diff)
		case <-ticker.C:
			storageWatcher.processQueue()
		}
	}
}

func (storageWatcher StorageWatcher) getTransformer(contractAddress common.Address) (transformer.StorageTransformer, bool) {
	return storageWatcher.transformerGetter(contractAddress)
}

func (storageWatcher StorageWatcher) processRow(diff utils.StorageDiff) {
	storageTransformer, ok := storageWatcher.getTransformer(diff.Contract)
	if !ok {
		logrus.Debug("ignoring a diff from an unwatched contract")
		return
	}
	executeErr := storageTransformer.Execute(diff)
	if executeErr != nil {
		logrus.Warn(fmt.Sprintf("error executing storage transformer: %s", executeErr))
		queueErr := storageWatcher.Queue.Add(diff)
		if queueErr != nil {
			logrus.Warn(fmt.Sprintf("error queueing storage diff: %s", queueErr))
		}
	}
}

func (storageWatcher StorageWatcher) processQueue() {
	diffs, fetchErr := storageWatcher.Queue.GetAll()
	if fetchErr != nil {
		logrus.Warn(fmt.Sprintf("error getting queued storage: %s", fetchErr))
	}
	for _, diff := range diffs {
		storageTransformer, ok := storageWatcher.getTransformer(diff.Contract)
		if !ok {
			// delete diff from queue if address no longer watched
			storageWatcher.deleteRow(diff.Id)
			continue
		}
		executeErr := storageTransformer.Execute(diff)
		if executeErr == nil {
			storageWatcher.deleteRow(diff.Id)
		}
	}
}

func (storageWatcher StorageWatcher) deleteRow(id int) {
	deleteErr := storageWatcher.Queue.Delete(id)
	if deleteErr != nil {
		logrus.Warn(fmt.Sprintf("error deleting persisted diff from queue: %s", deleteErr))
	}
}

func isKeyNotFound(executeErr error) bool {
	return reflect.TypeOf(executeErr) == reflect.TypeOf(utils.ErrStorageKeyNotFound{})
}
