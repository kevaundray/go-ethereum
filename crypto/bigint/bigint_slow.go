package bigint

import (
	"math/big"
)

type BigIntContext struct {
	modulus            *big.Int
	elementSizeInBytes uint

	scratchSpace []big.Int
}

// This is the maximum number of field elements that we will
// allow to be allocated in the scratch space at any particular time.
//
// Since the internal implementation could allow SIMD operations and the
// 128 bit SIMD is the least we should support. We use 256 to allow for
// 128 binary operations to happen at once; each operation uses 128 field
// elements.
const maxScratchSpaceSize = 256

// This is the maximum number of bytes that the chosen modulus can
// support.
// TODO: This number seems to be arbitrarily chosen as an upper bound.
const maxModulusSizeInBytes = 96

// ModFunc defines the type for modular arithmetic operations
type ModFunc func(*big.Int, *big.Int, *big.Int, *big.Int) *big.Int

// / New creates a BigIntContext which in turn will allow us to add integers
// / modulo the modulus passed in.
func NewBigIntContext(modBytes []byte, scratchSize int) (*BigIntContext, error) {
	if len(modBytes) > maxModulusSizeInBytes {
		return nil, errModulusTooBig
	}
	if len(modBytes) == 0 {
		return nil, errModulusMustBeNonZero
	}
	if scratchSize == 0 {
		return nil, errScratchSizeMustBeNonZero
	}
	if scratchSize > maxScratchSpaceSize {
		return nil, errScratchSpaceTooBig
	}

	return &BigIntContext{
		modulus:            new(big.Int).SetBytes(modBytes),
		elementSizeInBytes: uint(len(modBytes)),
		scratchSpace:       make([]big.Int, scratchSize),
	}, nil
}

func (b *BigIntContext) ElementSizeBytes() uint64 {
	return uint64(b.elementSizeInBytes)
}

// Store/Copy copies all of the bytes in `from` into the scratch space
func (b *BigIntContext) Store(dest, _count uint, from []byte) error {
	// Check that the from array is a multiple of the size of a field element
	if len(from)%int(b.elementSizeInBytes) != 0 {
		return errStoreWrongSize
	}
	numElements := len(from) / int(b.elementSizeInBytes)

	if uint(numElements) != _count {
		// TODO: This is a panic because we do not need both count and elementSize
		// TODO: Remove count and then this if statement in the future
		panic("count and numElements do not match")
	}

	// Check if the user tried to copy over nothing
	if len(from) == 0 {
		return errStoreCopyZero
	}

	// Check that there is not too much data, ie more data than scratch space
	maxNumScratchBytes := len(b.scratchSpace) * int(b.elementSizeInBytes)
	if len(from) > maxNumScratchBytes {
		return errStoreTooMuchData
	}

	// Check that the destination index allows for enough write space in the
	// scratch space
	freeScratchSpace := maxNumScratchBytes - int(dest)*int(b.elementSizeInBytes)
	if len(from) > freeScratchSpace {
		return errStoreInsufficientSpace
	}

	// Chunk the `from` slice into `elementSize` bytes and convert each chunk into
	// a field element to be stored on the stack
	scratchSpaceOffset := dest * b.elementSizeInBytes
	elementSizeInBytes := int(b.elementSizeInBytes)
	for i := 0; i < numElements; i++ {

		// First, lets get the `i'th` element from the
		// source array.
		start := i * elementSizeInBytes
		end := start + elementSizeInBytes
		sourceElement := new(big.Int).SetBytes(from[start:end])
		// Check if the integer is canonical
		cmp := sourceElement.Cmp(b.modulus)
		if cmp == 1 || cmp == 0 {
			return errElementNotCanonical
		}

		// Second, put it in the destination scratch space
		destinationPtr := int(scratchSpaceOffset) + i*int(b.elementSizeInBytes)
		b.scratchSpace[destinationPtr] = *sourceElement
	}

	return nil
}

func (b *BigIntContext) Load(dst []byte, fromPtr, count int) error {
	requiredSize := count * int(b.elementSizeInBytes)

	// Check that the buffer is big enough
	if len(dst) < requiredSize {
		return errDestinationBufferForLoadIsTooSmall
	}

	// Check that fromPtr and count do not access out of bounds
	if fromPtr < 0 || count < 0 || fromPtr+count > len(b.scratchSpace) {
		return errLoadOutOfBounds
	}

	// Copy elements from scratch space to destination buffer
	for i := 0; i < count; i++ {
		value := &b.scratchSpace[fromPtr+i]
		bytes := value.Bytes()

		// Pad with leading zeros if necessary to match element size
		padLen := int(b.elementSizeInBytes) - len(bytes)
		destOffset := i * int(b.elementSizeInBytes)

		if padLen > 0 {
			// Zero the full element size first
			for j := 0; j < int(b.elementSizeInBytes); j++ {
				dst[destOffset+j] = 0
			}
			// Copy the actual bytes after padding
			copy(dst[destOffset+padLen:], bytes)
		} else {
			// If no padding needed, just copy the bytes
			copy(dst[destOffset:], bytes)
		}
	}

	return nil
}

func (b *BigIntContext) MulMod(outPtr, outStride, xPtr, xStride, yPtr, yStride, count uint) error {
	return b.modularArithmetic(outPtr, outStride, xPtr, xStride, yPtr, yStride, count, mulMod)
}
func mulMod(out *big.Int, x *big.Int, y *big.Int, modulus *big.Int) *big.Int {
	return out.Mul(x, y).Mod(out, modulus)
}

func (b *BigIntContext) AddMod(outPtr, outStride, xPtr, xStride, yPtr, yStride, count uint) error {
	return b.modularArithmetic(outPtr, outStride, xPtr, xStride, yPtr, yStride, count, addMod)
}
func addMod(out *big.Int, x *big.Int, y *big.Int, modulus *big.Int) *big.Int {
	return out.Add(x, y).Mod(out, modulus)
}

func (b *BigIntContext) SubMod(outPtr, outStride, xPtr, xStride, yPtr, yStride, count uint) error {
	return b.modularArithmetic(outPtr, outStride, xPtr, xStride, yPtr, yStride, count, subMod)
}
func subMod(out *big.Int, x *big.Int, y *big.Int, modulus *big.Int) *big.Int {
	return out.Sub(x, y).Mod(out, modulus)
}

func (b *BigIntContext) modularArithmetic(outPtr, outStride, xPtr, xStride, yPtr, yStride, count uint, modFunc ModFunc) error {
	//  Check that the initial bounds are correct
	scratchSpaceSize := uint(len(b.scratchSpace))
	if outPtr >= scratchSpaceSize || xPtr >= scratchSpaceSize || yPtr >= scratchSpaceSize {
		return errOutOfBounds
	}

	if count == 0 {
		return errInvalidCountParameter
	}

	if xStride == 0 || outStride == 0 || yStride == 0 {
		return errCannotHaveZeroStride
	}

	// Check that end bounds are correct
	if count > 0 {
		lastOut := outPtr + (count-1)*outStride
		lastX := xPtr + (count-1)*xStride
		lastY := yPtr + (count-1)*yStride
		if lastOut >= scratchSpaceSize || lastX >= scratchSpaceSize || lastY >= scratchSpaceSize {
			return errOutOfBounds
		}
	}

	for i := uint(0); i < count; i++ {
		xSrc := xPtr + i*xStride
		ySrc := yPtr + i*yStride
		dst := outPtr + i*outStride

		out := &b.scratchSpace[dst]
		x := &b.scratchSpace[xSrc]
		y := &b.scratchSpace[ySrc]

		modFunc(out, x, y, b.modulus)
	}

	return nil
}
