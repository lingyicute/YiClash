package context

import (
	"github.com/lingyicute/yiclashcore/common/utils"
	"net"

	N "github.com/lingyicute/yiclashcore/common/net"
	C "github.com/lingyicute/yiclashcore/constant"

	"github.com/gofrs/uuid/v5"
)

type ConnContext struct {
	id       uuid.UUID
	metadata *C.Metadata
	conn     *N.BufferedConn
}

func NewConnContext(conn net.Conn, metadata *C.Metadata) *ConnContext {
	return &ConnContext{
		id:       utils.NewUUIDV4(),
		metadata: metadata,
		conn:     N.NewBufferedConn(conn),
	}
}

// ID implement C.ConnContext ID
func (c *ConnContext) ID() uuid.UUID {
	return c.id
}

// Metadata implement C.ConnContext Metadata
func (c *ConnContext) Metadata() *C.Metadata {
	return c.metadata
}

// Conn implement C.ConnContext Conn
func (c *ConnContext) Conn() *N.BufferedConn {
	return c.conn
}
