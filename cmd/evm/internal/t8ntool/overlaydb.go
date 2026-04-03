// Copyright 2024 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package t8ntool

import (
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/ethdb"
)

var errKeyNotFound = errors.New("not found")

// overlayDB wraps a read-only KeyValueStore with an in-memory write layer.
// Reads check the overlay first, then fall through to the underlying store.
// Writes and deletes go only to the overlay. The underlying store is never
// modified, making this safe for use with read-only databases.
type overlayDB struct {
	db      ethdb.KeyValueStore
	overlay map[string][]byte
	deleted map[string]struct{}
	mu      sync.RWMutex
}

func newOverlayDB(db ethdb.KeyValueStore) *overlayDB {
	return &overlayDB{
		db:      db,
		overlay: make(map[string][]byte),
		deleted: make(map[string]struct{}),
	}
}

func (o *overlayDB) Has(key []byte) (bool, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	k := string(key)
	if _, del := o.deleted[k]; del {
		return false, nil
	}
	if _, ok := o.overlay[k]; ok {
		return true, nil
	}
	return o.db.Has(key)
}

func (o *overlayDB) Get(key []byte) ([]byte, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	k := string(key)
	if _, del := o.deleted[k]; del {
		return nil, errKeyNotFound
	}
	if val, ok := o.overlay[k]; ok {
		return val, nil
	}
	return o.db.Get(key)
}

func (o *overlayDB) Put(key []byte, value []byte) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	k := string(key)
	delete(o.deleted, k)
	o.overlay[k] = value
	return nil
}

func (o *overlayDB) Delete(key []byte) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	k := string(key)
	delete(o.overlay, k)
	o.deleted[k] = struct{}{}
	return nil
}

func (o *overlayDB) DeleteRange(start, end []byte) error {
	// Not needed for t8n — just mark as unsupported
	return nil
}

func (o *overlayDB) NewBatch() ethdb.Batch {
	return &overlayBatch{db: o}
}

func (o *overlayDB) NewBatchWithSize(size int) ethdb.Batch {
	return &overlayBatch{db: o}
}

func (o *overlayDB) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	// Delegate to the underlying DB. This means iterators don't see
	// overlay writes, but for t8n's use case (trie iteration on the
	// committed state) this is acceptable.
	return o.db.NewIterator(prefix, start)
}

func (o *overlayDB) Stat() (string, error) {
	return o.db.Stat()
}

func (o *overlayDB) SyncKeyValue() error {
	return nil // no-op, nothing to sync
}

func (o *overlayDB) Compact(start []byte, limit []byte) error {
	return nil // no-op
}

func (o *overlayDB) Close() error {
	return o.db.Close()
}

// overlayBatch captures batch writes and applies them to the overlay.
type overlayBatch struct {
	db  *overlayDB
	ops []batchOp
}

type batchOp struct {
	key    []byte
	value  []byte
	delete bool
}

func (b *overlayBatch) Put(key []byte, value []byte) error {
	b.ops = append(b.ops, batchOp{
		key:   append([]byte{}, key...),
		value: append([]byte{}, value...),
	})
	return nil
}

func (b *overlayBatch) Delete(key []byte) error {
	b.ops = append(b.ops, batchOp{
		key:    append([]byte{}, key...),
		delete: true,
	})
	return nil
}

func (b *overlayBatch) ValueSize() int {
	size := 0
	for _, op := range b.ops {
		size += len(op.value)
	}
	return size
}

func (b *overlayBatch) Write() error {
	for _, op := range b.ops {
		if op.delete {
			b.db.Delete(op.key)
		} else {
			b.db.Put(op.key, op.value)
		}
	}
	return nil
}

func (b *overlayBatch) Reset() {
	b.ops = b.ops[:0]
}

func (b *overlayBatch) DeleteRange(start, end []byte) error {
	return nil
}

func (b *overlayBatch) Close() {
	b.ops = nil
}

func (b *overlayBatch) Replay(w ethdb.KeyValueWriter) error {
	for _, op := range b.ops {
		if op.delete {
			if err := w.Delete(op.key); err != nil {
				return err
			}
		} else {
			if err := w.Put(op.key, op.value); err != nil {
				return err
			}
		}
	}
	return nil
}
