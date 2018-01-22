package fluent

type bufferFullErr struct{}
type bufferFuller interface {
	BufferFull() bool
}
type causer interface {
	Cause() error
}

// Just need one instance
var bufferFullErrInstance bufferFullErr

// IsBufferFull returns true if the error is a BufferFull error
func IsBufferFull(e error) bool {
	for e != nil {
		if berr, ok := e.(bufferFuller); ok {
			return berr.BufferFull()
		}

		if cerr, ok := e.(causer); ok {
			e = cerr.Cause()
		}

		e = nil
	}
	return false
}

func (e *bufferFullErr) BufferFull() bool {
	return true
}

func (e *bufferFullErr) Error() string {
	return `buffer full`
}
