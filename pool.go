// pool.go - Only for internal buffer reuse
package iso8583

import "sync"

var bufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 4096)
		return &buf
	},
}

// Only pool buffers, not messages
func getBuffer() []byte {
	buf := bufferPool.Get().(*[]byte)
	return (*buf)[:0]
}

func putBuffer(buf []byte) {
	if cap(buf) <= 8192 { // Don't pool huge buffers
		b := buf[:0]
		bufferPool.Put(&b)
	}
}
