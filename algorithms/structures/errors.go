package structures

import "errors"

var (
	ErrListEmpty               = errors.New("list is empty")
	ErrIndexOutOfRange         = errors.New("index out of range")
	ErrQueueFull               = errors.New("queue overflow")
	ErrQueueEmpty              = errors.New("queue underflow")
	ErrQueueNegativeCapacity   = errors.New("negative queue capacity")
	ErrStackOverflow           = errors.New("stack overflow")
	ErrStackUnderflow          = errors.New("stack underflow")
	ErrStackNegativeCapacity   = errors.New("negative stack capacity")
	ErrHashTableInvalidBuckets = errors.New("hash table bucket count must be positive")
)
