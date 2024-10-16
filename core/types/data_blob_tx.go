package types

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/holiman/uint256"
	"github.com/protolambda/ztyp/codec"
	"github.com/protolambda/ztyp/conv"
	. "github.com/protolambda/ztyp/view"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

type ECDSASignature struct {
	V Uint8View
	R Uint256View
	S Uint256View
}

func (sig *ECDSASignature) Deserialize(dr *codec.DecodingReader) error {
	return dr.FixedLenContainer(&sig.V, &sig.R, &sig.S)
}

func (sig *ECDSASignature) Serialize(w *codec.EncodingWriter) error {
	return w.FixedLenContainer(&sig.V, &sig.R, &sig.S)
}

func (*ECDSASignature) ByteLength() uint64 {
	return 1 + 32 + 32
}

func (*ECDSASignature) FixedLength() uint64 {
	return 1 + 32 + 32
}

type AddressSSZ common.Address

func (addr *AddressSSZ) Deserialize(dr *codec.DecodingReader) error {
	if addr == nil {
		return errors.New("cannot deserialize into nil Address")
	}
	_, err := dr.Read(addr[:])
	return err
}

func (addr *AddressSSZ) Serialize(w *codec.EncodingWriter) error {
	return w.Write(addr[:])
}

func (*AddressSSZ) ByteLength() uint64 {
	return 20
}

func (*AddressSSZ) FixedLength() uint64 {
	return 20
}

// AddressOptionalSSZ implements Union[None, Address]
type AddressOptionalSSZ struct {
	Address *AddressSSZ
}

func (ao *AddressOptionalSSZ) Deserialize(dr *codec.DecodingReader) error {
	if ao == nil {
		return errors.New("cannot deserialize into nil Address")
	}
	if v, err := dr.ReadByte(); err != nil {
		return err
	} else if v == 0 {
		ao.Address = nil
		return nil
	} else if v == 1 {
		ao.Address = new(AddressSSZ)
		if err := ao.Address.Deserialize(dr); err != nil {
			return err
		}
		return nil
	} else {
		return fmt.Errorf("invalid union selector for Union[None, Address]: %d", v)
	}
}

func (ao *AddressOptionalSSZ) Serialize(w *codec.EncodingWriter) error {
	if ao.Address == nil {
		return w.WriteByte(0)
	} else {
		if err := w.WriteByte(1); err != nil {
			return err
		}
		return ao.Address.Serialize(w)
	}
}

func (ao *AddressOptionalSSZ) ByteLength() uint64 {
	if ao.Address == nil {
		return 1
	} else {
		return 1 + 20
	}
}

func (*AddressOptionalSSZ) FixedLength() uint64 {
	return 0
}

type TxDataView []byte

func (tdv *TxDataView) Deserialize(dr *codec.DecodingReader) error {
	return dr.ByteList((*[]byte)(tdv), MAX_CALLDATA_SIZE)
}

func (tdv TxDataView) Serialize(w *codec.EncodingWriter) error {
	return w.Write(tdv)
}

func (tdv TxDataView) ByteLength() (out uint64) {
	return uint64(len(tdv))
}

func (tdv *TxDataView) FixedLength() uint64 {
	return 0
}

func (tdv TxDataView) MarshalText() ([]byte, error) {
	return conv.BytesMarshalText(tdv[:])
}

func (tdv TxDataView) String() string {
	return "0x" + hex.EncodeToString(tdv[:])
}

func (tdv *TxDataView) UnmarshalText(text []byte) error {
	if tdv == nil {
		return errors.New("cannot decode into nil blob data")
	}
	return conv.DynamicBytesUnmarshalText((*[]byte)(tdv), text[:])
}

// ReadHashes reads length hashes from dr and returns them through hashes, reusing existing capacity if possible. Hashes will always be
// non-nil on return.
func ReadHashes(dr *codec.DecodingReader, hashes *[]common.Hash, length uint64) error {
	if uint64(len(*hashes)) != length {
		// re-use space if available (for recycling old state objects)
		if uint64(cap(*hashes)) >= length {
			*hashes = (*hashes)[:length]
		} else {
			*hashes = make([]common.Hash, length)
		}
	} else if *hashes == nil {
		// make sure the output is never nil
		*hashes = []common.Hash{}
	}
	dst := *hashes
	for i := uint64(0); i < length; i++ {
		if _, err := dr.Read(dst[i][:]); err != nil {
			return err
		}
	}
	return nil
}

func ReadHashesLimited(dr *codec.DecodingReader, hashes *[]common.Hash, limit uint64) error {
	scope := dr.Scope()
	if scope%32 != 0 {
		return fmt.Errorf("bad deserialization scope, cannot decode hashes list")
	}
	length := scope / 32
	if length > limit {
		return fmt.Errorf("too many hashes: %d > %d", length, limit)
	}
	return ReadHashes(dr, hashes, length)
}

func WriteHashes(ew *codec.EncodingWriter, hashes []common.Hash) error {
	for i := range hashes {
		if err := ew.Write(hashes[i][:]); err != nil {
			return err
		}
	}
	return nil
}

type VersionedHashesView []common.Hash

func (vhv *VersionedHashesView) Deserialize(dr *codec.DecodingReader) error {
	return ReadHashesLimited(dr, (*[]common.Hash)(vhv), MAX_VERSIONED_HASHES_LIST_SIZE)
}

func (vhv VersionedHashesView) Serialize(w *codec.EncodingWriter) error {
	return WriteHashes(w, vhv)
}

func (vhv VersionedHashesView) ByteLength() (out uint64) {
	return uint64(len(vhv)) * 32
}

func (vhv *VersionedHashesView) FixedLength() uint64 {
	return 0 // it's a list, no fixed length
}

type StorageKeysView []common.Hash

func (skv *StorageKeysView) Deserialize(dr *codec.DecodingReader) error {
	return ReadHashesLimited(dr, (*[]common.Hash)(skv), MAX_ACCESS_LIST_STORAGE_KEYS)
}

func (skv StorageKeysView) Serialize(w *codec.EncodingWriter) error {
	return WriteHashes(w, skv)
}

func (skv StorageKeysView) ByteLength() (out uint64) {
	return uint64(len(skv)) * 32
}

func (skv *StorageKeysView) FixedLength() uint64 {
	return 0 // it's a list, no fixed length
}

type AccessTupleView AccessTuple

func (atv *AccessTupleView) Deserialize(dr *codec.DecodingReader) error {
	return dr.Container((*AddressSSZ)(&atv.Address), (*StorageKeysView)(&atv.StorageKeys))
}

func (atv *AccessTupleView) Serialize(w *codec.EncodingWriter) error {
	return w.Container((*AddressSSZ)(&atv.Address), (*StorageKeysView)(&atv.StorageKeys))
}

func (atv *AccessTupleView) ByteLength() uint64 {
	return codec.ContainerLength((*AddressSSZ)(&atv.Address), (*StorageKeysView)(&atv.StorageKeys))
}

func (atv *AccessTupleView) FixedLength() uint64 {
	return 0
}

type AccessListView AccessList

func (alv *AccessListView) Deserialize(dr *codec.DecodingReader) error {
	*alv = AccessListView([]AccessTuple{})
	return dr.List(func() codec.Deserializable {
		i := len(*alv)
		*alv = append(*alv, AccessTuple{})
		return (*AccessTupleView)(&((*alv)[i]))
	}, 0, MAX_ACCESS_LIST_SIZE)
}

func (alv AccessListView) Serialize(w *codec.EncodingWriter) error {
	return w.List(func(i uint64) codec.Serializable {
		return (*AccessTupleView)(&alv[i])
	}, 0, uint64(len(alv)))
}

func (alv AccessListView) ByteLength() (out uint64) {
	for _, v := range alv {
		out += (*AccessTupleView)(&v).ByteLength() + codec.OFFSET_SIZE
	}
	return
}

func (alv *AccessListView) FixedLength() uint64 {
	return 0
}

type BlobTxMessage struct {
	ChainID          Uint256View
	Nonce            Uint64View
	GasTipCap        Uint256View // a.k.a. maxPriorityFeePerGas
	GasFeeCap        Uint256View // a.k.a. maxFeePerGas
	Gas              Uint64View
	To               AddressOptionalSSZ // nil means contract creation
	Value            Uint256View
	Data             TxDataView
	AccessList       AccessListView
	MaxFeePerDataGas Uint256View

	BlobVersionedHashes VersionedHashesView
}

func (tx *BlobTxMessage) Deserialize(dr *codec.DecodingReader) error {
	return dr.Container(&tx.ChainID, &tx.Nonce, &tx.GasTipCap, &tx.GasFeeCap, &tx.Gas, &tx.To, &tx.Value, &tx.Data, &tx.AccessList, &tx.MaxFeePerDataGas, &tx.BlobVersionedHashes)
}

func (tx *BlobTxMessage) Serialize(w *codec.EncodingWriter) error {
	return w.Container(&tx.ChainID, &tx.Nonce, &tx.GasTipCap, &tx.GasFeeCap, &tx.Gas, &tx.To, &tx.Value, &tx.Data, &tx.AccessList, &tx.MaxFeePerDataGas, &tx.BlobVersionedHashes)
}

func (tx *BlobTxMessage) ByteLength() uint64 {
	return codec.ContainerLength(&tx.ChainID, &tx.Nonce, &tx.GasTipCap, &tx.GasFeeCap, &tx.Gas, &tx.To, &tx.Value, &tx.Data, &tx.AccessList, &tx.MaxFeePerDataGas, &tx.BlobVersionedHashes)
}

func (tx *BlobTxMessage) FixedLength() uint64 {
	return 0
}

func (tx *BlobTxMessage) setChainID(chainID *big.Int) {
	(*uint256.Int)(&tx.ChainID).SetFromBig(chainID)
}

func (stx *SignedBlobTx) ByteLength() uint64 {
	return codec.ContainerLength(&stx.Message, &stx.Signature)
}

func (stx *SignedBlobTx) FixedLength() uint64 {
	return 0
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *BlobTxMessage) copy() *BlobTxMessage {
	cpy := &BlobTxMessage{
		ChainID:             tx.ChainID,
		Nonce:               tx.Nonce,
		GasTipCap:           tx.GasTipCap,
		GasFeeCap:           tx.GasFeeCap,
		Gas:                 tx.Gas,
		To:                  AddressOptionalSSZ{Address: (*AddressSSZ)(copyAddressPtr((*common.Address)(tx.To.Address)))},
		Value:               tx.Value,
		Data:                common.CopyBytes(tx.Data),
		AccessList:          make([]AccessTuple, len(tx.AccessList)),
		MaxFeePerDataGas:    tx.MaxFeePerDataGas,
		BlobVersionedHashes: make([]common.Hash, len(tx.BlobVersionedHashes)),
	}
	copy(cpy.AccessList, tx.AccessList)
	copy(cpy.BlobVersionedHashes, tx.BlobVersionedHashes)

	return cpy
}

type SignedBlobTx struct {
	Message   BlobTxMessage
	Signature ECDSASignature
}

const (
	MAX_CALLDATA_SIZE              = 1 << 24
	MAX_ACCESS_LIST_SIZE           = 1 << 24
	MAX_ACCESS_LIST_STORAGE_KEYS   = 1 << 24
	MAX_VERSIONED_HASHES_LIST_SIZE = 1 << 24
)

func (stx *SignedBlobTx) Deserialize(dr *codec.DecodingReader) error {
	return dr.Container(&stx.Message, &stx.Signature)
}

func (stx *SignedBlobTx) Serialize(w *codec.EncodingWriter) error {
	return w.Container(&stx.Message, &stx.Signature)
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (stx *SignedBlobTx) copy() TxData {
	cpy := &SignedBlobTx{
		Message:   *stx.Message.copy(),
		Signature: stx.Signature,
	}

	return cpy
}

func u256ToBig(v *Uint256View) *big.Int {
	return (*uint256.Int)(v).ToBig()
}

// accessors for innerTx.
func (stx *SignedBlobTx) txType() byte               { return BlobTxType }
func (stx *SignedBlobTx) chainID() *big.Int          { return u256ToBig(&stx.Message.ChainID) }
func (stx *SignedBlobTx) accessList() AccessList     { return AccessList(stx.Message.AccessList) }
func (stx *SignedBlobTx) dataHashes() []common.Hash  { return stx.Message.BlobVersionedHashes }
func (stx *SignedBlobTx) data() []byte               { return stx.Message.Data }
func (stx *SignedBlobTx) gas() uint64                { return uint64(stx.Message.Gas) }
func (stx *SignedBlobTx) gasFeeCap() *big.Int        { return u256ToBig(&stx.Message.GasFeeCap) }
func (stx *SignedBlobTx) gasTipCap() *big.Int        { return u256ToBig(&stx.Message.GasTipCap) }
func (stx *SignedBlobTx) maxFeePerDataGas() *big.Int { return u256ToBig(&stx.Message.MaxFeePerDataGas) }
func (stx *SignedBlobTx) gasPrice() *big.Int         { return u256ToBig(&stx.Message.GasFeeCap) }
func (stx *SignedBlobTx) value() *big.Int            { return u256ToBig(&stx.Message.Value) }
func (stx *SignedBlobTx) nonce() uint64              { return uint64(stx.Message.Nonce) }
func (stx *SignedBlobTx) to() *common.Address        { return (*common.Address)(stx.Message.To.Address) }

func (stx *SignedBlobTx) rawSignatureValues() (v, r, s *big.Int) {
	return big.NewInt(int64(stx.Signature.V)), u256ToBig(&stx.Signature.R), u256ToBig(&stx.Signature.S)
}

func (stx *SignedBlobTx) setSignatureValues(chainID, v, r, s *big.Int) {
	stx.Message.setChainID(chainID)
	stx.Signature.V = Uint8View(v.Uint64())
	(*uint256.Int)(&stx.Signature.R).SetFromBig(r)
	(*uint256.Int)(&stx.Signature.S).SetFromBig(s)
}

func (tx *SignedBlobTx) effectiveGasPrice(dst *big.Int, baseFee *big.Int) *big.Int {
	if baseFee == nil {
		return dst.Set(tx.gasFeeCap())
	}
	tip := dst.Sub(tx.gasFeeCap(), baseFee)
	if tip.Cmp(tx.gasTipCap()) > 0 {
		tip.Set(tx.gasTipCap())
	}
	return tip.Add(tip, baseFee)
}

// fakeExponential approximates factor * e ** (num / denom) using a taylor expansion
// as described in the EIP-4844 spec.
func fakeExponential(factor, num, denom *big.Int) *big.Int {
	output := new(big.Int)
	numAccum := new(big.Int).Mul(factor, denom)
	for i := 1; numAccum.Sign() > 0; i++ {
		output.Add(output, numAccum)
		numAccum.Mul(numAccum, num)
		iBig := big.NewInt(int64(i))
		numAccum.Div(numAccum, iBig.Mul(iBig, denom))
	}
	return output.Div(output, denom)
}

// GetDataGasPrice implements get_data_gas_price from EIP-4844
func GetDataGasPrice(excessDataGas *big.Int) *big.Int {
	if excessDataGas == nil {
		return nil
	}
	return fakeExponential(big.NewInt(params.MinDataGasPrice), excessDataGas, big.NewInt(params.DataGasPriceUpdateFraction))
}

// GetDataGasUsed returns the amount of datagas consumed by a transaction with the specified number
// of blobs
func GetDataGasUsed(blobs int) uint64 {
	return uint64(blobs) * params.DataGasPerBlob
}
