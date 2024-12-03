package vm

import (
	"errors"

	"github.com/ethereum/go-ethereum/crypto/bigint"
)

// The EVM will allocate a set of BigInt contexts, one for each modulus that
// the user wants to support. This structure will be used to store all of
// the contexts for these modulis.
type BigIntCtxAllocations struct {
	allocatedContexts map[uint]*bigint.BigIntContext // TODO: can we use a vector instead of map? the id would be automatically assigned and returned on the stack
	currentContext    *bigint.BigIntContext
	allocedSize       uint64
}

func newBigIntCtxAllocations() *BigIntCtxAllocations {
	return &BigIntCtxAllocations{
		allocatedContexts: make(map[uint]*bigint.BigIntContext),
		currentContext:    &bigint.BigIntContext{},
		allocedSize:       0,
	}
}

func (b *BigIntCtxAllocations) AllocateContextAndSetAsCurrent(id uint, modulus []byte, allocSize int) error {
	fieldContext, err := bigint.NewBigIntContext(modulus, allocSize)
	if err != nil {
		return err
	}

	b.allocatedContexts[id] = fieldContext
	b.currentContext = fieldContext
	b.allocedSize += uint64(allocSize)

	return nil
}

func (b *BigIntCtxAllocations) ChangeContext(id uint) error {
	if id >= uint(len(b.allocatedContexts)) {
		return errors.New("invalid big integer context id ")
	}
	fieldContext := b.allocatedContexts[id]
	b.currentContext = fieldContext
	return nil
}
