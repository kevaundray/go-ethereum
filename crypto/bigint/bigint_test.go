package bigint

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"testing"
)

// This is just a simple smoke test, can delete after
func TestBigIntContext_New(t *testing.T) {
	tests := []struct {
		name        string
		modBytes    []byte
		scratchSize int
		wantErr     error
	}{
		{
			name:        "Valid modulus and scratch size",
			modBytes:    []byte{0x01, 0x02, 0x03},
			scratchSize: 128,
			wantErr:     nil,
		},
		{
			name:        "Empty modulus",
			modBytes:    []byte{},
			scratchSize: 128,
			wantErr:     errModulusMustBeNonZero,
		},
		{
			name:        "Zero scratch size",
			modBytes:    []byte{0x01},
			scratchSize: 0,
			wantErr:     errScratchSizeMustBeNonZero,
		},
		{
			name:        "Exceeds max scratch size",
			modBytes:    []byte{0x01},
			scratchSize: maxScratchSpaceSize + 1,
			wantErr:     errScratchSpaceTooBig,
		},
		{
			name:        "Modulus too large",
			modBytes:    make([]byte, maxModulusSizeInBytes+1),
			scratchSize: 128,
			wantErr:     errModulusTooBig,
		},
		{
			name:        "Maximum allowed modulus size",
			modBytes:    make([]byte, maxModulusSizeInBytes),
			scratchSize: 128,
			wantErr:     nil,
		},
		{
			name:        "Maximum allowed scratch size",
			modBytes:    []byte{0x01},
			scratchSize: maxScratchSpaceSize,
			wantErr:     nil,
		},
	}

	for _, tt := range tests {
		got, err := NewBigIntContext(tt.modBytes, tt.scratchSize)

		// Check if we should have gotten an error
		if tt.wantErr != nil {
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != nil {
				t.Fatal("returned non-nil BigIntContext when error occurred")
			}
			return
		}

		if err != nil {
			fmt.Println(tt.wantErr)
			t.Fatalf("unexpected error = %v", err)
		}

		if got == nil {
			t.Fatal("returned nil BigIntContext")
		}

		if got.modulus == nil {
			t.Fatal("returned BigIntContext with nil modulus")
		}

		// Check if the modulus bytes match the input
		if !bytes.Equal(got.modulus.Bytes(), tt.modBytes) {
			// Check if its all zeroes, because big.Int will
			// return an empty vector
			if !allZeroes(tt.modBytes) || !slices.Equal(got.modulus.Bytes(), []byte{}) {
				t.Fatalf("got modulus bytes = %v, want %v in test `%s`", got.modulus.Bytes(), tt.modBytes, tt.name)
			}
		}
	}
}

func TestBigIntContext_ModularArithmetic(t *testing.T) {
	// Helper function to create a BigIntContext with initialized scratch space
	setupContext := func(modulus []byte, scratchVals []uint64) (*BigIntContext, error) {
		ctx, err := NewBigIntContext(modulus, len(scratchVals))
		if err != nil {
			return nil, err
		}

		for i, val := range scratchVals {
			ctx.scratchSpace[i].SetUint64(val)
		}

		return ctx, nil
	}

	tests := []struct {
		name        string
		modulus     []byte
		scratch     []uint64
		outPtr      uint
		outStride   uint
		xPtr        uint
		xStride     uint
		yPtr        uint
		yStride     uint
		count       uint
		operation   string // "mul", "add", or "sub"
		wantErr     error
		wantResults []uint64
	}{
		{
			name:        "MulMod: Simple multiplication",
			modulus:     []byte{17},
			scratch:     []uint64{0, 4, 5},
			outPtr:      0,
			outStride:   1,
			xPtr:        1,
			xStride:     1,
			yPtr:        2,
			yStride:     1,
			count:       1,
			operation:   "mul",
			wantErr:     nil,
			wantResults: []uint64{3}, // (4 * 5) mod 17 = 3
		},
		{
			name:        "AddMod: Simple addition",
			modulus:     []byte{17},
			scratch:     []uint64{0, 4, 5},
			outPtr:      0,
			outStride:   1,
			xPtr:        1,
			xStride:     1,
			yPtr:        2,
			yStride:     1,
			count:       1,
			operation:   "add",
			wantErr:     nil,
			wantResults: []uint64{9}, // (4 + 5) mod 17 = 9
		},
		{
			name:        "SubMod: Simple subtraction",
			modulus:     []byte{17},
			scratch:     []uint64{0, 15, 4},
			outPtr:      0,
			outStride:   1,
			xPtr:        1,
			xStride:     1,
			yPtr:        2,
			yStride:     1,
			count:       1,
			operation:   "sub",
			wantErr:     nil,
			wantResults: []uint64{11}, // (15 - 4) mod 17 = 11
		},
		{
			name:        "Multiple operations with stride",
			modulus:     []byte{17},
			scratch:     []uint64{0, 4, 5, 0, 6, 7},
			outPtr:      0,
			outStride:   3,
			xPtr:        1,
			xStride:     3,
			yPtr:        2,
			yStride:     3,
			count:       2,
			operation:   "mul",
			wantErr:     nil,
			wantResults: []uint64{3, 8}, // [(4 * 5) mod 17, (6 * 7) mod 17]
		},
		{
			name:      "Zero count parameter",
			modulus:   []byte{17},
			scratch:   []uint64{0, 1, 2},
			outPtr:    0,
			outStride: 1,
			xPtr:      1,
			xStride:   1,
			yPtr:      2,
			yStride:   1,
			count:     0,
			operation: "mul",
			wantErr:   errInvalidCountParameter,
		},
		{
			name:      "Out of bounds with stride",
			modulus:   []byte{17},
			scratch:   []uint64{0, 1, 2},
			outPtr:    3,
			outStride: 2,
			xPtr:      0,
			xStride:   1,
			yPtr:      1,
			yStride:   1,
			count:     2,
			operation: "mul",
			wantErr:   errOutOfBounds,
		},
		{
			name:        "Large modular arithmetic",
			modulus:     []byte{0xFF, 0xFF}, // 65535
			scratch:     []uint64{0, 65000, 1000},
			outPtr:      0,
			outStride:   1,
			xPtr:        1,
			xStride:     1,
			yPtr:        2,
			yStride:     1,
			count:       1,
			operation:   "add",
			wantErr:     nil,
			wantResults: []uint64{465}, // (65000 + 1000) mod 65535 = 465
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := setupContext(tt.modulus, tt.scratch)
			if err != nil {
				t.Fatalf("Failed to setup context: %v", err)
			}

			var opErr error
			switch tt.operation {
			case "mul":
				opErr = ctx.MulMod(tt.outPtr, tt.outStride, tt.xPtr, tt.xStride, tt.yPtr, tt.yStride, tt.count)
			case "add":
				opErr = ctx.AddMod(tt.outPtr, tt.outStride, tt.xPtr, tt.xStride, tt.yPtr, tt.yStride, tt.count)
			case "sub":
				opErr = ctx.SubMod(tt.outPtr, tt.outStride, tt.xPtr, tt.xStride, tt.yPtr, tt.yStride, tt.count)
			default:
				t.Fatalf("unrecognized operation %v", tt.operation)
			}

			if tt.wantErr != nil {
				if !errors.Is(opErr, tt.wantErr) {
					t.Fatalf("Operation error = %v, wantErr %v", opErr, tt.wantErr)
				}
				return
			}

			if opErr != nil {
				t.Errorf("Unexpected error = %v", opErr)
				return
			}

			// Verify results
			for i, want := range tt.wantResults {
				outPos := tt.outPtr + uint(i)*tt.outStride
				got := ctx.scratchSpace[outPos].Uint64()
				if got != want {
					t.Errorf("Result at position %d = %d, want %d", i, got, want)
				}
			}
		})
	}
}

func TestBigIntContext_Store(t *testing.T) {
	tests := []struct {
		name        string
		modBytes    []byte
		dest        uint
		count       uint
		from        []byte
		wantErr     error
		scratchSize int
		checkVals   []uint64
	}{
		{
			name:        "Simple storage",
			modBytes:    []byte{17},
			dest:        0,
			count:       1,
			from:        []byte{0x05},
			wantErr:     nil,
			scratchSize: 3,
			checkVals:   []uint64{5},
		},
		{
			name:        "Empty input",
			modBytes:    []byte{0x11},
			dest:        0,
			count:       0,
			from:        []byte{},
			wantErr:     errStoreCopyZero,
			scratchSize: 3,
		},
		{
			name:        "Non-canonical element",
			modBytes:    []byte{0x11}, // modulus = 17
			dest:        0,
			count:       1,
			from:        []byte{0x11}, // 17 >= 17 (modulus)
			wantErr:     errElementNotCanonical,
			scratchSize: 3,
		},
		{
			name:        "Input not multiple of element size",
			modBytes:    []byte{0x11, 0x00}, // 2-byte elements
			dest:        0,
			count:       1,
			from:        []byte{0x05}, // Only 1 byte
			wantErr:     errStoreWrongSize,
			scratchSize: 3,
		},
		{
			name:        "Too much data for scratch space",
			modBytes:    []byte{0x11},
			dest:        0,
			count:       5,
			from:        []byte{0x01, 0x02, 0x03, 0x04, 0x05},
			wantErr:     errStoreTooMuchData,
			scratchSize: 3,
		},
		{
			name:        "Insufficient space at destination",
			modBytes:    []byte{0x11},
			dest:        2,
			count:       2,
			from:        []byte{0x05, 0x07},
			wantErr:     errStoreInsufficientSpace,
			scratchSize: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, err := NewBigIntContext(tt.modBytes, tt.scratchSize)
			if err != nil {
				t.Fatalf("Failed to create context: %v", err)
			}

			err = ctx.Store(tt.dest, tt.count, tt.from)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("Store() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("Store() unexpected error = %v", err)
				return
			}

			// Verify stored values
			if tt.checkVals != nil {
				for i, expected := range tt.checkVals {
					got := ctx.scratchSpace[int(tt.dest)+i].Uint64()
					if got != expected {
						t.Errorf("Store() element at index %d = %d, want %d", i, got, expected)
					}
				}
			}
		})
	}
}

func TestBigIntContext_Load(t *testing.T) {
	tests := []struct {
		name      string
		modBytes  []byte   // Modulus to set up context
		setupVals []uint64 // Values to store in scratch space
		fromPtr   int      // Where to load from
		count     int      // How many elements to load
		dstSize   int      // Size of destination buffer
		wantErr   error
		wantBytes []byte // Expected loaded bytes
	}{
		{
			name:      "Simple load single element",
			modBytes:  []byte{0x11}, // modulus = 17
			setupVals: []uint64{5},  // Single value in scratch space
			fromPtr:   0,
			count:     1,
			dstSize:   1, // 1 byte per element
			wantErr:   nil,
			wantBytes: []byte{0x05},
		},
		{
			name:      "Load multiple elements",
			modBytes:  []byte{0x11},       // modulus = 17
			setupVals: []uint64{5, 7, 10}, // Three values
			fromPtr:   0,
			count:     3,
			dstSize:   3, // 1 byte per element * 3 elements
			wantErr:   nil,
			wantBytes: []byte{0x05, 0x07, 0x0A},
		},
		{
			name:      "Load with offset",
			modBytes:  []byte{0x11},
			setupVals: []uint64{5, 7, 10},
			fromPtr:   1, // Start from second element
			count:     2,
			dstSize:   2,
			wantErr:   nil,
			wantBytes: []byte{0x07, 0x0A},
		},
		{
			name:      "Destination buffer too small",
			modBytes:  []byte{0x11},
			setupVals: []uint64{5, 7},
			fromPtr:   0,
			count:     2,
			dstSize:   1, // Too small for 2 elements
			wantErr:   errDestinationBufferForLoadIsTooSmall,
			wantBytes: nil,
		},
		{
			name:      "Source out of bounds",
			modBytes:  []byte{0x11},
			setupVals: []uint64{5, 7},
			fromPtr:   1,
			count:     2, // Would read beyond scratch space
			dstSize:   2,
			wantErr:   errLoadOutOfBounds,
			wantBytes: nil,
		},
		{
			name:      "Negative fromPtr",
			modBytes:  []byte{0x11},
			setupVals: []uint64{5, 7},
			fromPtr:   -1,
			count:     1,
			dstSize:   1,
			wantErr:   errLoadOutOfBounds,
			wantBytes: nil,
		},
		{
			name:      "Large elements",
			modBytes:  []byte{0xFF, 0xFF}, // 2-byte modulus
			setupVals: []uint64{0x0102, 0x0304},
			fromPtr:   0,
			count:     2,
			dstSize:   4, // 2 bytes per element * 2 elements
			wantErr:   nil,
			wantBytes: []byte{0x01, 0x02, 0x03, 0x04},
		},
	}

	for _, tt := range tests {
		// Create context and set up scratch space
		ctx, err := NewBigIntContext(tt.modBytes, len(tt.setupVals))
		if err != nil {
			t.Fatalf("Failed to create context: %v", err)
		}

		// Set up scratch space with test values
		for i, val := range tt.setupVals {
			ctx.scratchSpace[i].SetUint64(val)
		}

		// Create destination buffer
		dst := make([]byte, tt.dstSize)

		// Perform load operation
		err = ctx.Load(dst, tt.fromPtr, tt.count)

		// Check error
		if tt.wantErr != nil {
			if err != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
			return
		}

		if err != nil {
			t.Errorf("Load() unexpected error = %v", err)
			return
		}

		// Verify loaded bytes
		if !bytes.Equal(dst, tt.wantBytes) {
			t.Errorf("Load() got bytes = %v, want %v", dst, tt.wantBytes)
		}
	}
}

func allZeroes(byts []byte) bool {
	for _, byt := range byts {
		if byt != 0 {
			return false
		}
	}

	return true
}
