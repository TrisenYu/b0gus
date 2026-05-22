package services

// SPDX-LICENSE-IDENTIFIER: BSD 3-Clause License

import (
	"b0gus/configs"
	"b0gus/crypto_aux"
	"bytes"
	"context"
	"crypto/tls"
	"net"
	"sync/atomic"
)

// PullUpdatesFromRemote will start up itself as a locality trust TLS server on configs.RemotePullSource.
// The TLS server will cancel once the in-param ctx is canceled.
//
// Currently, only used in pem_op_test.go
//
//nolint:unused
func PullUpdatesFromRemote(
	rootCaPath, signedCertPath, signedKeyPath string,
	ctx context.Context,
) {
	// listen at local port and obey some formal syntax/private protocol
	// during the runtime, the monitored port might be altered to another port
	// so the session

	// notice that this interface will provide distributed communication ability
	// so encryption is required
	// var clientAux = NetType{}
	// clientAux.Init(configs.RawEnum)
	// ConcurrentEventDispatcher(&clientAux, )
	// clientAux.AlterNetFd(TCPEnum)
	tmpConf := crypto_aux.LoadLocalCertAsTLSServ(rootCaPath, signedCertPath, signedKeyPath)
	if tmpConf == nil {
		configs.Logger().Error("empty tlsConfig!")
		return
	}
	listener, err := tls.Listen(
		"tcp", configs.RemotePullSource,
		tmpConf,
	)
	if err != nil || listener == nil {
		configs.Logger().Warn("can not listen on given tls port...")
		if err != nil {
			configs.Logger().Error(err.Error())
		} else {
			configs.Logger().Error("listener is nil")
		}
		return
	}
	defer func() { _ = listener.Close() }()
	var atomFlag atomic.Bool
	atomFlag.Store(true)
	go func() {
		<-ctx.Done()
		atomFlag.Store(false)
		_ = listener.Close()
	}()
	for atomFlag.Load() {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		// TODO: 0. also requires limitor for connections control.
		// 		 1. handle in another go routine.
		// 		 2. updates from different businesses should queue and then attempt to apply,
		//			distribute responses of error if the provided updates is unacceptable.
		go handleTrustUpdateClients(conn)
	}
}

// Currently, only used in pem_op_test.go
//
//nolint:unused
func handleTrustUpdateClients(tlsConn net.Conn) {
	defer func() { _ = tlsConn.Close() }()
	for {
		buf := make([]byte, 1024)
		_, err := tlsConn.Read(buf)
		if err != nil {
			configs.Logger().Warn(err.Error())
			return
		}
		// we have to deal with a private protocol
		// otherwise we can only collect limited string at one time
		configs.Logger().Info(string(bytes.Trim(buf, "\x00")))
	}
}
