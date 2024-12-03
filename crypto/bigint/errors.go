package bigint

import "errors"

var (
	errInvalidCountParameter    = errors.New("count parameter cannot be zero")
	errModulusTooBig            = errors.New("modulus exceeds maximum allowed size")
	errOutOfBounds              = errors.New("pointers are out of bounds")
	errModulusMustBeNonZero     = errors.New("modulus must be non-zero")
	errScratchSizeMustBeNonZero = errors.New("scratch size must have non-zero, to allow for us field element arithmetic")
	errScratchSpaceTooBig       = errors.New("scratch space cannot exceed maximum scratch space size")
	errCannotHaveZeroStride     = errors.New("cannot have a zero stride")

	errStoreWrongSize         = errors.New("store can only copy in data that is a multiple of the element size")
	errStoreCopyZero          = errors.New("cannot copy zero bytes")
	errStoreTooMuchData       = errors.New("cannot copy more bytes than there is scratch space")
	errStoreInsufficientSpace = errors.New("cannot copy more bytes than there is available scratch space")
	errElementNotCanonical    = errors.New("field element must be canonical")

	errDestinationBufferForLoadIsTooSmall = errors.New("destination buffer for load operation is too small")
	errLoadOutOfBounds                    = errors.New("load operation would access memory out of bounds")
)
