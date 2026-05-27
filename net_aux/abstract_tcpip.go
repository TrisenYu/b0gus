package net_aux

/// Last modified at 2026/05/16 星期六 12:21:03
// SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License

import (
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/quic-go/quic-go/http3"
	"golang.org/x/net/icmp"

	"b0gus/configs"
	"b0gus/crypto_aux"
)

func (nt *ReentrantNetType) AlterNetFd(
	netType NetTypeEnum,
	confObj *atomic.Pointer[configs.LocalConfig],
) {
	tag, _ := nt.getCallBackFnWithServTag()
	if tag < configs.RawEnum || tag >= configs.ENDofEnum {
		configs.Logger().Error(
			configs.GetLocalizedMsg("services.absTCPIPUninitErr", nil),
		)
		return
	} else if confObj == nil {
		configs.Logger().Error(
			configs.GetLocalizedMsg("services.absTCPIPUnknownNetType", nil),
		)
		return
	}
	switch netType {
	case TCPEnum: // re-entry
		if nt.servFd != nil {
			nt.CloseServFd()
		}
		nt.tcp(confObj)
	case UDPEnum: // re-entry
		if nt.servFd != nil {
			nt.CloseServFd()
		}
		nt.udp(confObj)
	case HTTPEnum: // re-entry
		if nt.servFd != nil {
			nt.CloseServFd()
		}
		nt.http(confObj)
	case ICMPEnum: // flip-flop
		// because there is no notion named port in ICMP.
		// for windows, skip this function calling.
		if runtime.GOOS == "windows" {
			configs.Logger().Warn(
				configs.GetLocalizedMsg("services.absTCPIPunsupportedOSOnICMPErr", nil),
			)
			return
		}
		if nt.servFd != nil {
			nt.CloseServFd()
		} else {
			nt.icmp()
		}
	default:
		configs.Logger().Warn(
			configs.GetLocalizedMsg("services.UnknownNetType", nil),
		)
		return
	}
}

func (nt *ReentrantNetType) errHandler(err error) {
	tag, _ := nt.getCallBackFnWithServTag()
	choice, ok := configs.ServLUT[tag]
	if !ok {
		choice = "<unknown services>: "
	}
	var sb strings.Builder
	sb.WriteString(choice)
	sb.WriteString(err.Error())
	configs.Logger().Warn(sb.String())
}

func (nt *ReentrantNetType) portFailCheck(port uint16) bool {
	if port > 1024 {
		return false
	}
	var sb strings.Builder
	sb.WriteString(strconv.Itoa(int(port)))
	sb.WriteString(" - selected privilege port for serving is less than 1024")
	nt.errHandler(errors.New(sb.String()))
	return true
}

func (nt *ReentrantNetType) tcp(ConfObj *atomic.Pointer[configs.LocalConfig]) {
	tag, _ := nt.getCallBackFnWithServTag()
	port := ConfObj.Load().SelectPort(tag)
	if nt.portFailCheck(port) {
		return
	}
	var (
		listener net.Listener = nil
		err      error
		sb       strings.Builder
	)
	// [TODO]: extended definitions of self-signed-off certificate for intranet?
	certPath, keyPath, RemoteServ := ConfObj.Load().SelectTLSpairWithRemoteHost(tag)
	sb.WriteRune(':')
	sb.WriteString(strconv.Itoa(int(port)))
	if len(certPath) != 0 && len(keyPath) != 0 {
		tlsConf := (*tls.Config)(nil)
		if len(RemoteServ) == 0 {
			tlsConf = crypto_aux.LoadNormalCertAsTLSServ(certPath, keyPath)
		} else {
			tlsConf = crypto_aux.LoadNormalCertAsTLSClient(certPath, keyPath, RemoteServ)
		}
		if tlsConf != nil {
			listener, err = tls.Listen("tcp", sb.String(), tlsConf)
		}
	}
	if listener == nil {
		listener, err = net.Listen("tcp", sb.String())
	}
	if err != nil {
		nt.errHandler(err)
		return
	}
	defer func() { _ = listener.Close() }()

	nt.fdGuard.Lock()
	nt.servFd = listener
	nt.fdGuard.Unlock()

	nt.clientLimiter.Store(0)
	defer nt.CloseServFd()
	nt.ShutdownFlag.Store(false)
	for !nt.ShutdownFlag.Load() {
		nt.fdGuard.RLock()
		currListener, ok := nt.servFd.(net.Listener)
		if !ok {
			break
		}
		nt.fdGuard.RUnlock()
		inConn, err := currListener.Accept()
		if err != nil {
			// [FIXME]: noisy for the same connection try
			nt.errHandler(err)
			continue
		}
		if nt.ShutdownFlag.Load() {
			_ = inConn.Close()
			_ = currListener.Close()
			break
		}
		go nt.tcpClientHandler(inConn, ConfObj)
	}
}

func (nt *ReentrantNetType) tcpClientHandler(
	inConn net.Conn, ConfObj *atomic.Pointer[configs.LocalConfig],
) {
	tag, _ := nt.getCallBackFnWithServTag()
	if nt.clientLimiter.Load() >= ConfObj.Load().SelectMaxClient(tag) {
		_ = inConn.Close()
		return
	}
	nt.clientLimiter.Add(1)
	defer func() {
		_ = inConn.Close()
		nt.clientLimiter.Add(^uint32(0))
	}()
	timeoutVal := ConfObj.Load().SelectTimeout(tag)
	if timeoutVal != 0 {
		currTimeout := time.Duration(timeoutVal) * time.Second
		err := inConn.SetDeadline(time.Now().Add(currTimeout))
		if err != nil {
			nt.errHandler(err)
			return
		}
	}
	_, cb := nt.getCallBackFnWithServTag()
	if cb == nil {
		return
	}
	cb.InvokeForTCPtask(inConn)
}

// CloseServFd attempt to close nt.servFd and set it to nil when finding it is not nil.
func (nt *ReentrantNetType) CloseServFd() {
	nt.fdGuard.Lock()
	defer nt.fdGuard.Unlock()
	if nt.servFd != nil {
		_ = nt.servFd.Close()
		nt.servFd = nil
	}
}

func (nt *ReentrantNetType) udp(ConfObj *atomic.Pointer[configs.LocalConfig]) {
	tag, _ := nt.getCallBackFnWithServTag()
	port := ConfObj.Load().SelectPort(tag)
	if nt.portFailCheck(port) {
		return
	}
	udpListener, err := net.ListenUDP(
		"udp", &net.UDPAddr{
			IP:   net.IPv6loopback,
			Port: int(port),
		},
	)
	if err != nil {
		nt.errHandler(err)
		return
	}
	defer func() {
		_ = udpListener.Close()
		nt.CloseServFd()
	}()
	nt.fdGuard.Lock()
	nt.servFd = udpListener
	nt.fdGuard.Unlock()

	var checkFlag atomic.Bool
	checkFlag.Store(false)
	for !checkFlag.Load() {
		nt.fdGuard.RLock()
		currListener, ok := nt.servFd.(*net.UDPConn)
		if !ok {
			configs.Logger().Warn(
				configs.GetLocalizedMsg("services.UDPConnCastingErr", nil),
			)
			nt.fdGuard.RUnlock()
			break
		}
		nt.fdGuard.RUnlock()
		// Don't set timeout for udp connection
		dataBuf := make([]byte, 1024)
		n, remoteConn, err := currListener.ReadFromUDP(dataBuf)
		dataBuf = dataBuf[:n]
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe) {
			break
		} else if err != nil {
			nt.errHandler(err)
			continue
		}
		go nt.udpResponse(remoteConn, dataBuf)
	}
}

func (nt *ReentrantNetType) udpResponse(addr *net.UDPAddr, dataBuf []byte) {
	wConn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		nt.errHandler(err)
		return
	}
	defer func() { _ = wConn.Close() }()
	_, cb := nt.getCallBackFnWithServTag()
	if cb == nil {
		return
	}
	_, _ = wConn.Write(cb.InvokeForUDPtask(addr, dataBuf))
}

// icmp implements for Internet Control Message Protocol
func (nt *ReentrantNetType) icmp() {
	icmpListener, err := icmp.ListenPacket("udp4", "localhost")
	if err != nil {
		nt.errHandler(err)
		return
	}
	nt.fdGuard.Lock()
	nt.servFd = icmpListener
	nt.fdGuard.Unlock()
	defer func() { _ = icmpListener.Close() }()
	var checkFlag atomic.Bool
	checkFlag.Store(false)

	for !checkFlag.Load() {
		dataBuf := make([]byte, 1024)
		length, remoteConn, err := icmpListener.ReadFrom(dataBuf)
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrClosedPipe) {
			break
		} else if err != nil {
			nt.errHandler(err)
			continue
		}
		_, cb := nt.getCallBackFnWithServTag()
		if cb == nil {
			continue
		}
		/*
			// something works like this:
			var sb strings.Builder
			sb.WriteString("receive message from <")
			sb.WriteString(srcIP.String())
			sb.WriteString(">, length is ")
			sb.WriteString(strconv.Itoa(len(remotePayload)))
			configs.Logger().Info(sb.String())
		*/
		go cb.InvokeForICMPtask(remoteConn, dataBuf[:length])
	}
}

func (nt *ReentrantNetType) http(confObj *atomic.Pointer[configs.LocalConfig]) {
	tag, cb := nt.getCallBackFnWithServTag()
	port := confObj.Load().SelectPort(tag)
	if nt.portFailCheck(port) {
		return
	} else if cb == nil {
		// in this case, show as a try for meaningless HTTP listening
		return
	}
	certPath, keyPath, _ := confObj.Load().SelectTLSpairWithRemoteHost(tag)
	var (
		err error
		sb  strings.Builder
	)

	sb.WriteRune(':')
	sb.WriteString(strconv.Itoa(int(port)))
	if len(certPath) == 0 || len(keyPath) == 0 {
		listener := &http.Server{
			Addr:    sb.String(),
			Handler: cb,
		}
		nt.fdGuard.Lock()
		nt.servFd = listener
		nt.fdGuard.Unlock()
		defer func() { _ = listener.Close() }()
		err = listener.ListenAndServe()
	} else {
		listener := &http3.Server{
			Addr:    sb.String(),
			Handler: cb,
		}
		nt.fdGuard.Lock()
		nt.servFd = listener
		nt.fdGuard.Unlock()
		defer func() { _ = listener.Close() }()
		err = listener.ListenAndServeTLS(certPath, keyPath)
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		nt.errHandler(err)
	}
}

// Init requires pass `port`, `callback`, `clientNumThreshold`, `clientConnTimeout`.
// In order to identify with which service the ReentrantNetType is helping,
// servTag is thereby required.
func (nt *ReentrantNetType) Init(servTag configs.ServEnum, callBack AbsServNetAux) {
	nt.cbGuard.Lock()
	defer nt.cbGuard.Unlock()
	nt.servTag, nt.callBack = servTag, callBack
}

func (nt *ReentrantNetType) getCallBackFnWithServTag() (configs.ServEnum, AbsServNetAux) {
	nt.cbGuard.RLock()
	defer nt.cbGuard.RUnlock()
	return nt.servTag, nt.callBack
}

// EventMonitor will attempt to update or shutdown the networking port
func (nt *ReentrantNetType) EventMonitor(
	confObj *atomic.Pointer[configs.LocalConfig],
	scc *configs.ServConcurrentCtrl,
) {
	tag, cb := nt.getCallBackFnWithServTag()
	if tag < configs.RawEnum || cb == nil {
		return
	}
back:
	select {
	case <-scc.TerminatedCtx.Done(): // terminated notification
	case netTypeStr := <-scc.ServNetTypeCh:
		var choice = TCPEnum
		tmp := strings.ToLower(netTypeStr)
		if strings.Contains(tmp, "udp") {
			choice = UDPEnum
		} else if strings.Contains(tmp, "icmp") {
			choice = ICMPEnum
		} else if strings.Contains(tmp, "http") {
			choice = HTTPEnum
		}
		nt.AlterNetFd(choice, confObj)
		goto back
	}
	nt.ShutdownFlag.Store(true)
	nt.CloseServFd()
}
