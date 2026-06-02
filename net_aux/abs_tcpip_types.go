package net_aux

import (
	"b0gus/configs"
	"io"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
)

// AbsServNetAux is a business-callback interface used by the ReentrantNetType.
type AbsServNetAux interface {
	http.Handler

	// InvokeForTCPtask uses for executing TCP-typed Business
	InvokeForTCPtask(conn net.Conn)

	// InvokeForUDPtask uses for executing UDP-typed Business
	InvokeForUDPtask(addr net.Addr, payload []byte) []byte

	// InvokeForICMPtask uses for executing ICMP-typed Business,
	// merely on **unix-system**
	InvokeForICMPtask(addr net.Addr, payload []byte)
}

// ReentrantNetType is used for directly setup network listener without caring
// how to update monitoring port or its corresponding type.
type ReentrantNetType struct {
	fdGuard  sync.RWMutex
	cbGuard  sync.RWMutex // cbGuard is used for promising the concurrent security of callBack
	callBack AbsServNetAux
	// clientLimiter is as a filter to limit the max number of alive connections.
	clientLimiter atomic.Uint32
	ShutdownFlag  atomic.Bool
	servFd        io.Closer // servFd temporarily stores the listener and is guarded by fdGuard
	// servTag holds the service ID.
	servTag configs.ServEnum
}

type NetTypeEnum int // indicates the specific tcp/ip suite

const (
	TCPEnum NetTypeEnum = iota
	UDPEnum
	HTTPEnum // actually http here is quic
	ICMPEnum
	OtherEnum
)
