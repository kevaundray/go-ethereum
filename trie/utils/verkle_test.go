// Copyright 2023 go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package utils

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/gballet/go-verkle"
	"github.com/holiman/uint256"
)

func TestTreeKeyAddress0(t *testing.T) {
	var (
		address = []byte{0x00}
	)

	expectedVersionKey := "bf101a6e1c8e83c11bd203a582c7981b91097ec55cbd344ce09005c1f26d1900"
	if expectedVersionKey != hex.EncodeToString(VersionKey(address)) {
		t.Fatal("Unmatched version key")
	}
}

func TestTreeKeyAddressSmoke(t *testing.T) {
	var (
		address = []byte{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 10,
			11, 12, 13, 14, 15, 16, 17, 18, 19, 20,
			21, 22, 23, 24, 25, 26, 27, 28, 29, 30,
			31, 32}
		treeIndex = []byte{
			33, 34, 35, 36, 37, 38, 39, 40,
			41, 42, 43, 44, 45, 46, 47, 48, 49, 50,
			51, 52, 53, 54, 55, 56, 57, 58, 59, 60,
			61, 62, 63, 64}
		subIndex = byte(0)
	)

	treeIndexU256 := uint256.NewInt(0)
	treeIndexU256.SetBytes32(treeIndex)

	expectedHex := "76a014d14e338c57342cda5187775c6b75e7f0ef292e81b176c7a5a700273700"

	gotBytes := GetTreeKey(address, treeIndexU256, subIndex)
	gotHex := hex.EncodeToString(gotBytes)

	if expectedHex != gotHex {
		t.Fatalf("Unmatched tree key: expected %s, got %s", expectedHex, gotHex)
	}
}

func TestTreeKeyAddressBesuInteropDelete(t *testing.T) {
	var (
		subIndex = byte(0)
	)

	address, err := hex.DecodeString("003f9549040250ec5cdef31947e5213edee80ad2d5bba35c9e48246c5d9213d6")
	if err != nil {
		t.Fatalf("Failed to decode address: %v", err)
	}

	treeIndexBytes, err := hex.DecodeString("004C6CE0115457AC1AB82968749EB86ED2D984743D609647AE88299989F91271")
	if err != nil {
		t.Fatalf("Failed to decode tree index: %v", err)
	}

	treeIndexU256 := uint256.NewInt(0)
	treeIndexU256.SetBytes32(treeIndexBytes)

	expectedHex := "ff6e8f1877fd27f91772a4cec41d99d2f835d7320e929b8d509c5fa7ce095c00"

	gotBytes := GetTreeKey(address, treeIndexU256, subIndex)
	gotHex := hex.EncodeToString(gotBytes)

	if expectedHex != gotHex {
		t.Fatalf("Unmatched tree key: expected %s, got %s", expectedHex, gotHex)
	}
}

func TestTreeKey(t *testing.T) {
	var (
		address      = []byte{0x01}
		addressEval  = evaluateAddressPoint(address)
		smallIndex   = uint256.NewInt(1)
		largeIndex   = uint256.NewInt(10000)
		smallStorage = []byte{0x1}
		largeStorage = bytes.Repeat([]byte{0xff}, 16)
	)
	if !bytes.Equal(VersionKey(address), VersionKeyWithEvaluatedAddress(addressEval)) {
		t.Fatal("Unmatched version key")
	}
	if !bytes.Equal(BalanceKey(address), BalanceKeyWithEvaluatedAddress(addressEval)) {
		t.Fatal("Unmatched balance key")
	}
	if !bytes.Equal(NonceKey(address), NonceKeyWithEvaluatedAddress(addressEval)) {
		t.Fatal("Unmatched nonce key")
	}
	if !bytes.Equal(CodeKeccakKey(address), CodeKeccakKeyWithEvaluatedAddress(addressEval)) {
		t.Fatal("Unmatched code keccak key")
	}
	if !bytes.Equal(CodeSizeKey(address), CodeSizeKeyWithEvaluatedAddress(addressEval)) {
		t.Fatal("Unmatched code size key")
	}
	if !bytes.Equal(CodeChunkKey(address, smallIndex), CodeChunkKeyWithEvaluatedAddress(addressEval, smallIndex)) {
		t.Fatal("Unmatched code chunk key")
	}
	if !bytes.Equal(CodeChunkKey(address, largeIndex), CodeChunkKeyWithEvaluatedAddress(addressEval, largeIndex)) {
		t.Fatal("Unmatched code chunk key")
	}
	if !bytes.Equal(StorageSlotKey(address, smallStorage), StorageSlotKeyWithEvaluatedAddress(addressEval, smallStorage)) {
		t.Fatal("Unmatched storage slot key")
	}
	if !bytes.Equal(StorageSlotKey(address, largeStorage), StorageSlotKeyWithEvaluatedAddress(addressEval, largeStorage)) {
		t.Fatal("Unmatched storage slot key")
	}
}

// goos: darwin
// goarch: amd64
// pkg: github.com/ethereum/go-ethereum/trie/utils
// cpu: VirtualApple @ 2.50GHz
// BenchmarkTreeKey
// BenchmarkTreeKey-8   	  398731	      2961 ns/op	      32 B/op	       1 allocs/op
func BenchmarkTreeKey(b *testing.B) {
	// Initialize the IPA settings which can be pretty expensive.
	verkle.GetConfig()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		BalanceKey([]byte{0x01})
	}
}

// goos: darwin
// goarch: amd64
// pkg: github.com/ethereum/go-ethereum/trie/utils
// cpu: VirtualApple @ 2.50GHz
// BenchmarkTreeKeyWithEvaluation
// BenchmarkTreeKeyWithEvaluation-8   	  513855	      2324 ns/op	      32 B/op	       1 allocs/op
func BenchmarkTreeKeyWithEvaluation(b *testing.B) {
	// Initialize the IPA settings which can be pretty expensive.
	verkle.GetConfig()

	addr := []byte{0x01}
	eval := evaluateAddressPoint(addr)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BalanceKeyWithEvaluatedAddress(eval)
	}
}

// goos: darwin
// goarch: amd64
// pkg: github.com/ethereum/go-ethereum/trie/utils
// cpu: VirtualApple @ 2.50GHz
// BenchmarkStorageKey
// BenchmarkStorageKey-8   	  230516	      4584 ns/op	      96 B/op	       3 allocs/op
func BenchmarkStorageKey(b *testing.B) {
	// Initialize the IPA settings which can be pretty expensive.
	verkle.GetConfig()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		StorageSlotKey([]byte{0x01}, bytes.Repeat([]byte{0xff}, 32))
	}
}

// goos: darwin
// goarch: amd64
// pkg: github.com/ethereum/go-ethereum/trie/utils
// cpu: VirtualApple @ 2.50GHz
// BenchmarkStorageKeyWithEvaluation
// BenchmarkStorageKeyWithEvaluation-8   	  320125	      3753 ns/op	      96 B/op	       3 allocs/op
func BenchmarkStorageKeyWithEvaluation(b *testing.B) {
	// Initialize the IPA settings which can be pretty expensive.
	verkle.GetConfig()

	addr := []byte{0x01}
	eval := evaluateAddressPoint(addr)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		StorageSlotKeyWithEvaluatedAddress(eval, bytes.Repeat([]byte{0xff}, 32))
	}
}
