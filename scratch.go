package lzfse

import "sync"

// scratchPool recycles scratch buffers across calls. The native routines only
// use the scratch as workspace, so buffers are reused without clearing.
var scratchPool sync.Pool

type scratchBuffer struct{ b []byte }

// getScratch returns a workspace of size bytes and the function that returns
// it to the pool. A size of zero means the routine needs no workspace.
func (lib *Library) getScratch(size int) (uintptr, func(), error) {
	if size <= 0 {
		return 0, func() {}, nil
	}
	var buf []byte
	if v := scratchPool.Get(); v != nil {
		sb := v.(*scratchBuffer)
		if cap(sb.b) >= size {
			buf = sb.b[:size]
		}
	}
	if buf == nil {
		buf = make([]byte, size)
	}
	owned := buf
	return sliceData(owned), func() {
		scratchPool.Put(&scratchBuffer{b: owned[:cap(owned)]})
	}, nil
}
