package config

import (
	"github.com/lingyicute/yiclashcore/listener/sing"

	"encoding/json"
)

type ShadowsocksServer struct {
	Enable    bool
	Listen    string
	Password  string
	Cipher    string
	Udp       bool
	MuxOption sing.MuxOption `yaml:"mux-option" json:"mux-option,omitempty"`
}

func (t ShadowsocksServer) String() string {
	b, _ := json.Marshal(t)
	return string(b)
}
