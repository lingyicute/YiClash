package nat

import (
	"net"

	"github.com/lingyicute/yiclashcore/common/atomic"
	C "github.com/lingyicute/yiclashcore/constant"
)

type writeBackProxy struct {
	wb atomic.TypedValue[C.WriteBack]
}

func (w *writeBackProxy) WriteBack(b []byte, addr net.Addr) (n int, err error) {
	return w.wb.Load().WriteBack(b, addr)
}

func (w *writeBackProxy) UpdateWriteBack(wb C.WriteBack) {
	w.wb.Store(wb)
}

func NewWriteBackProxy(wb C.WriteBack) C.WriteBackProxy {
	w := &writeBackProxy{}
	w.UpdateWriteBack(wb)
	return w
}
