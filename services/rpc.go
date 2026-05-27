package services

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"math/rand/v2"
	"net"
	"strconv"
	"strings"
	"sync/atomic"

	"b0gus/configs"
	"b0gus/crypto_aux"

	consulApi "github.com/hashicorp/consul/api/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type b0gusRPCTaskaServ struct {
	UnimplementedTaskCheckServer
	atomConf *atomic.Pointer[configs.LocalConfig]
	cancel   context.CancelFunc
}

//nolint:unused
func (b *b0gusRPCTaskaServ) Start(
	ctx context.Context,
	request *ServicesRequest,
) (*ServicesResponse, error) {
	// request.ServEnum
	var tag configs.ServEnum
	ret := &ServicesResponse{
		Ok:  ServicesResponse_InternalErr,
		Msg: "Start: unexpected internal service",
	}
	switch request.ServEnum {
	case ServicesRequest_SSH:
		tag = configs.SSHEnum
	case ServicesRequest_DNS:
		tag = configs.DNSEnum
	case ServicesRequest_HTTP:
		tag = configs.HTTPEnum
	case ServicesRequest_NTP:
		tag = configs.NTPEnum
	case ServicesRequest_SMTP:
		tag = configs.SMTPEnum
	default:
		tag = configs.ENDofEnum
	}
	functor := tagMapsToServ[tag]
	if functor == nil {
		return ret, errors.New(ret.Msg)
	}
	childCtx, cancel := context.WithCancel(ctx)
	b.cancel = cancel
	// [TODO]
	// 		1. recover configuration from request.Config
	// 		2. control the execution of functor.
	if b.atomConf == nil {
		ret.Ok = ServicesResponse_InternalErr
		ret.Msg = "Start: no config for this service"
		return ret, errors.New(ret.Msg)
	}

	var parsedStruct configs.LocalConfig
	buf := bytes.NewReader(request.Payload)
	err := binary.Read(buf, binary.LittleEndian, &parsedStruct)
	if err != nil {
		ret.Ok = ServicesResponse_InternalErr
		ret.Msg = err.Error()
		return ret, errors.New(ret.Msg)
	}
	b.atomConf.Store(&parsedStruct)
	go functor(b.atomConf, &configs.ServConcurrentCtrl{TerminatedCtx: childCtx}, genericArgs(tag, b.atomConf))
	var sb strings.Builder
	sb.WriteString("Start: ")
	sb.WriteString(configs.ServLUT[tag])
	sb.WriteString(" has started")
	ret.Ok = ServicesResponse_OK
	ret.Msg = sb.String()
	return ret, nil
}

//nolint:unused
func (b *b0gusRPCTaskaServ) Terminate(
	ctx context.Context,
	request *ServicesRequest,
) (*ServicesResponse, error) {
	res := &ServicesResponse{
		Ok:  ServicesResponse_RequestErr,
		Msg: "No available cancel function is set",
	}
	if b.cancel != nil {
		b.cancel()
		res.Msg = "Operation OK"
		res.Ok = ServicesResponse_OK
	}
	return res, nil
}

//nolint:unused
func (b *b0gusRPCTaskaServ) UpdateConf(
	ctx context.Context,
	request *ServicesRequest,
) (*ServicesResponse, error) {
	res := &ServicesResponse{
		Ok:  ServicesResponse_InternalErr,
		Msg: "No available atomConf is set for rpc service",
	}
	if b.atomConf == nil || b.atomConf.Load() == nil {
		return res, errors.New(res.Msg)
	} else if len(request.Payload) != binary.Size(configs.LocalConfig{}) {
		res.Ok = ServicesResponse_RequestErr
		res.Msg = "empty configuration provided for current rpc service"
		return res, errors.New(res.Msg)
	}
	// load configuration data from request
	var parsedStruct configs.LocalConfig
	buf := bytes.NewReader(request.Payload)
	err := binary.Read(buf, binary.LittleEndian, &parsedStruct)
	if err != nil {
		res.Msg = err.Error()
		return res, err
	}
	b.atomConf.Store(&parsedStruct)
	res.Ok = ServicesResponse_OK
	res.Msg = "Update OK"
	return res, nil
}

//nolint:unused
func tmpSetupRpcServ(localServPort int, FilePair configs.KeyCertPathPair) {
	var sb strings.Builder
	sb.WriteRune(':')
	sb.WriteString(strconv.Itoa(localServPort))
	listener, err := net.Listen("tcp", sb.String())
	sb.Reset()
	if err != nil {
		configs.Logger().Error(err.Error())
		return
	}
	defer func() { _ = listener.Close() }()
	creds, err := credentials.NewServerTLSFromFile(
		FilePair.GetCertPath(), FilePair.GetKeyPath(),
	)
	if err != nil {
		configs.Logger().Error(err.Error())
		return
	}
	grpcServer := grpc.NewServer(grpc.Creds(creds))
	RegisterTaskCheckServer(
		grpcServer,
		&b0gusRPCTaskaServ{
			atomConf: &atomic.Pointer[configs.LocalConfig]{},
		},
	)
	err = grpcServer.Serve(listener)
	if err != nil {
		configs.Logger().Error(err.Error())
		return
	}
}

//nolint:unused
func tmpInternalClient(
	rpcServAddrStr string,
	tlsFilePair configs.KeyCertPathPair,
) {
	opt := make([]grpc.DialOption, 0)
	if tlsFilePair.SelfCheck() {
		tlsConf := crypto_aux.LoadNormalCertAsTLSClient(
			tlsFilePair.GetCertPath(), tlsFilePair.GetKeyPath(), rpcServAddrStr,
		)
		if tlsConf == nil {
			configs.Logger().Error("unable to load tls for rpc client")
			return
		}
		opt = append(opt, grpc.WithTransportCredentials(credentials.NewTLS(tlsConf)))
	} else {
		opt = append(opt, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	conn, err := grpc.NewClient(rpcServAddrStr, opt...)
	if err != nil {
		configs.Logger().Error(err.Error())
		return
	}
	defer func() { _ = conn.Close() }()
	cli := NewTaskCheckClient(conn)
	_, err = cli.Start(context.Background(), nil)
	_, err = cli.Terminate(context.Background(), nil)
}

type rpcServiceInfo struct {
	ID   string `toml:"id" json:"id" mapstructure:"id"`
	Name string `toml:"name" json:"name" mapstructure:"name"`
	Addr string `toml:"addr" json:"addr" mapstructure:"addr" validate:"ip|hostname_rfc1123"`
	Port int    `toml:"port" json:"port" mapstructure:"port" validate:"port"`
}

//nolint:unused
func tmpNewInternalConsulRpcCli(tlsFilePath configs.KeyCertPathPair) *consulApi.Client {
	tlsConf := crypto_aux.LoadNormalCertAsTLSServ(tlsFilePath.GetCertPath(), tlsFilePath.GetKeyPath())
	if tlsConf == nil {
		// log error
		return nil
	}
	conf := consulApi.DefaultConfig()
	if tlsFilePath.SelfCheck() {
		conf.TLSConfig.CertFile = tlsFilePath.GetCertPath()
		conf.TLSConfig.KeyFile = tlsFilePath.GetKeyPath()
	}
	cli, err := consulApi.NewClient(conf)
	if err != nil {
		configs.Logger().Error(err.Error())
		return nil
	}
	return cli
}

//nolint:unused
func tmpRpcServiceRegisterToConsul(
	cli *consulApi.Client,
	rpcServ rpcServiceInfo,
) error {
	service := &consulApi.AgentServiceRegistration{
		ID:      rpcServ.ID,
		Name:    rpcServ.Name,
		Address: rpcServ.Addr,
		Port:    rpcServ.Port,
	}
	return cli.Agent().ServiceRegister(service)
}

//nolint:unused
func tmpDiscoveryRpcService(
	cli *consulApi.Client,
	servName string,
) (servAddr string, err error) {
	servInstances, _, err := cli.Health().Service(servName, "", true, nil)
	if err != nil {
		return
	}
	if len(servInstances) < 1 {
		err = errors.New("no service found")
		return
	}
	idx := rand.IntN(len(servInstances))
	instance := servInstances[idx].Service
	var sb strings.Builder
	sb.WriteString(instance.Address)
	sb.WriteString(":")
	sb.WriteString(strconv.Itoa(instance.Port))
	servAddr = sb.String()
	return
}
