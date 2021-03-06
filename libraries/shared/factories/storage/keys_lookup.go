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

package storage

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/makerdao/vulcanizedb/libraries/shared/storage"
	"github.com/makerdao/vulcanizedb/pkg/datastore/postgres"
)

type KeysLookup interface {
	Lookup(key common.Hash) (storage.ValueMetadata, error)
	SetDB(db *postgres.DB)
}

type keysLookup struct {
	loader   KeysLoader
	mappings map[common.Hash]storage.ValueMetadata
}

func NewKeysLookup(loader KeysLoader) KeysLookup {
	return &keysLookup{loader: loader, mappings: make(map[common.Hash]storage.ValueMetadata)}
}

func (lookup *keysLookup) Lookup(key common.Hash) (storage.ValueMetadata, error) {
	metadata, ok := lookup.mappings[key]
	if !ok {
		refreshErr := lookup.refreshMappings()
		if refreshErr != nil {
			return metadata, refreshErr
		}
		metadata, ok = lookup.mappings[key]
		if !ok {
			return metadata, storage.ErrKeyNotFound{Key: key.Hex()}
		}
	}
	return metadata, nil
}

func (lookup *keysLookup) refreshMappings() error {
	var err error
	lookup.mappings, err = lookup.loader.LoadMappings()
	if err != nil {
		return err
	}
	lookup.mappings = storage.AddHashedKeys(lookup.mappings)
	return nil
}

func (lookup *keysLookup) SetDB(db *postgres.DB) {
	lookup.loader.SetDB(db)
}
