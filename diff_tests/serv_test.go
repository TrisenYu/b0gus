package diff_tests

import (
	"b0gus/configs"
	"b0gus/crypto_aux"
	"b0gus/internal/mock"
	"b0gus/services"
	"io"
	"math/rand/v2"
	"net"
	"os"
	"strconv"

	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/golang/mock/gomock"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

type metaMocker struct {
	ctrl     *gomock.Controller
	conf     configs.LocalConfig
	FakeConf atomic.Pointer[configs.LocalConfig]
	// context-related
	Ctx    context.Context
	Cancel context.CancelFunc
}

type mockEvaluator struct {
	metaMocker
	// mock-related
	MockDB *mock.MockDBhandler
}

func (m *mockEvaluator) ReloadConf(t *testing.T) {
	err := gofakeit.Struct(&m.conf)
	assert.NoError(t, err)
	m.FakeConf.Store(&m.conf)
}

func (m *mockEvaluator) ReloadByAssignment(payload any) {
	switch p := payload.(type) {
	case configs.SSHconfig:
		m.conf.ServerConfig.SSHconfig = p
	case configs.SMTPconfig:
		m.conf.ServerConfig.SMTPconfig = p
	case configs.HTTPconfig:
		m.conf.ServerConfig.HTTPconfig = p
	case configs.NTPconfig:
		m.conf.ServerConfig.NTPconfig = p
	case configs.DNSconfig:
		m.conf.ServerConfig.DNSconfig = p
	default:
		return
	}
	m.FakeConf.Store(&m.conf)
}

func (m *mockEvaluator) Init(t *testing.T, timeout time.Duration) {
	m.initCtrl(t)
	m.Ctx, m.Cancel = context.WithTimeout(context.Background(), timeout)
	m.MockDB = mock.NewMockDBhandler(m.ctrl)
	m.MockDB.EXPECT().AlterDatabaseHandler(gomock.Any()).Return().AnyTimes()
	m.MockDB.EXPECT().CreateTable(gomock.Any()).Return(nil).AnyTimes()
	m.MockDB.EXPECT().CreateOrUpdateItemsInSeq(gomock.Any()).Return(nil).AnyTimes()
	m.MockDB.EXPECT().CreateOrUpdateItem(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
}

func (m *mockEvaluator) Free() {
	m.FinishCtrl()
	if m.Cancel != nil {
		m.Cancel()
	}
}

func (m *mockEvaluator) initCtrl(t *testing.T) {
	m.FinishCtrl()
	m.ctrl = gomock.NewController(t)
}

// FinishCtrl will finish the initiated gomock.Controller.
func (m *mockEvaluator) FinishCtrl() {
	if m.ctrl != nil {
		m.ctrl.Finish()
	}
}

// Access function helps to fetch the sub struct from the preset LocalConfig.
func (m *mockEvaluator) Access(enumTag configs.ServEnum) any {
	val := m.conf.SelectTerm(enumTag)
	if val != nil {
		return val
	}
	return configs.ServInit[enumTag]
}

func TestMockSSH(t *testing.T) {
	var (
		pemPath = "./test_dir/test.pem"
		pemType = "ed25519"
	)

	m := mockEvaluator{}
	m.Init(t, 15*time.Second)
	defer m.Free()

	delPem := func() {
		err := os.Remove(pemPath)
		assert.Equal(t, nil, err, "Unable to delete test.pem")
	}
	defer delPem()
	servKey := crypto_aux.LoadOrCreateSSHpem(pemPath, pemType, 0)
	assert.NotNil(t, servKey, "Still got an nil after creating")

	m.ReloadConf(t)
	val, ok := m.Access(configs.SSHEnum).(configs.SSHconfig)
	assert.True(t, ok, "Expected to find SSH config")
	val.TLSKeyPath = pemPath
	val.PermitLogin = true
	val.ListenPort = 3222
	m.ReloadByAssignment(val)
	restore := zap.ReplaceGlobals(zap.NewNop())
	defer restore()
	go (&services.SSHServConf{}).Run(
		&m.FakeConf,
		&configs.ServConcurrentCtrl{Ctx: m.Ctx},
		m.MockDB, servKey,
	)
	sshConf := &ssh.ClientConfig{
		User: gofakeit.Username(),
		Auth: []ssh.AuthMethod{
			ssh.Password(
				gofakeit.Password(true, true, true, true, false, 7),
			),
		},
		HostKeyCallback: ssh.FixedHostKey(servKey.PublicKey()),
	}
	client, err := ssh.Dial("tcp", "localhost:"+strconv.Itoa(int(val.ListenPort)), sshConf)
	assert.NoError(t, err)
	if client == nil {
		m.Ctx.Done()
		t.Fatal("unable to use a client")
		return
	}
	defer func() { _ = client.Close() }()
	session, err := client.NewSession()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer func() { _ = session.Close() }()
	session.Stderr, session.Stdout = io.Discard, io.Discard
	_ = session.RequestPty("xterm", 80, 40, nil)
	stdinPipe, err := session.StdinPipe()
	if err != nil {
		t.Fatal(err)
		return
	}
	defer func() { _ = stdinPipe.Close() }()
	_ = session.Shell()
	for _, cmd := range CmdSeeds {
		_, _ = stdinPipe.Write([]byte(cmd))
		_ = session.WindowChange(rand.IntN(1024), rand.IntN(1024))
	}
	<-m.Ctx.Done()
}

func TestMockNTP(t *testing.T) {
	m := mockEvaluator{}
	m.Init(t, 15*time.Second)
	defer m.Free()

	m.ReloadConf(t)
	val, ok := m.Access(configs.NTPEnum).(configs.NTPconfig)
	assert.True(t, ok, "Expected to find NTP config")
	m.ReloadByAssignment(val)

	// [TODO]: Noisy and chaos logging actions
	restore := zap.ReplaceGlobals(zap.NewNop())
	defer restore()
	go (&services.NTPServConf{}).Run(
		&m.FakeConf,
		&configs.ServConcurrentCtrl{Ctx: m.Ctx},
		m.MockDB, nil,
	)

	conn, err := net.Dial("udp", "localhost:"+strconv.Itoa(int(val.ListenPort)))
	assert.NoError(t, err)
	if conn == nil {
		m.Cancel()
		t.Fatal("unable to use UDP connection")
		return
	}
	defer func() { _ = conn.Close() }()
	for _, cmd := range CmdSeeds {
		// write only UDP connection
		_, _ = conn.Write([]byte(cmd))
	}
	<-m.Ctx.Done()
}

var (
	dnsTestcase = []struct {
		domain  string
		dnsType uint16
	}{
		{"www.baidu.com", dns.TypeA},
		{"api.baidu.com", dns.TypeAAAA},
		{"cname.baidu.com", dns.TypeCNAME},
		{"ptr.baidu.com", dns.TypePTR},
		{"_acme.xyz.com", dns.TypeTXT},
	}
)

func TestMockDNS(t *testing.T) {
	m := mockEvaluator{}
	m.Init(t, 15*time.Second)
	defer m.Free()

	m.ReloadConf(t)
	val, ok := m.Access(configs.DNSEnum).(configs.DNSconfig)
	assert.True(t, ok, "Expected to find DNS config")
	val.ListenPort = 5553
	m.ReloadByAssignment(val)
	defer m.Cancel()
	restore := zap.ReplaceGlobals(zap.NewNop())
	defer restore()
	go (&services.DNSservConf{}).Run(
		&m.FakeConf,
		&configs.ServConcurrentCtrl{Ctx: m.Ctx},
		m.MockDB, nil,
	)
	c := dns.Client{
		Net:     "udp",
		Timeout: 2 * time.Second,
	}
	for _, testcase := range dnsTestcase {
		dm := new(dns.Msg)
		dm.SetQuestion(dns.Fqdn(testcase.domain), testcase.dnsType)
		dm.RecursionDesired = true
		in, _, err := c.Exchange(dm, "localhost:"+strconv.Itoa(int(val.ListenPort)))
		assert.NoError(t, err)
		if err != nil {
			m.Cancel()
			t.Fatalf("unable to use UDP connection for DNS server. err: %v", err)
			return
		}
		if in.Rcode != dns.RcodeSuccess {
			t.Fatalf("DNS Fault: %s", dns.RcodeToString[in.Rcode])
			return
		}
	}
	<-m.Ctx.Done()
}

//func TestMockHTTP(t *testing.T) {
//	m := mockEvaluator{}
//	m.Init(t, 15*time.Second)
//	defer m.Free()
//
//	m.ReloadConf(t)
//	val, ok := m.Access(configs.HTTPEnum).(configs.HTTPconfig)
//  assert.True(t, ok, "Expected to find HTTP config")
//	m.ReloadByAssignment(val)
//	defer m.Cancel()
//
//	go (&services.HTTPservConf{}).Run(
//		&m.FakeConf,
//		&configs.ServConcurrentCtrl{Ctx: m.Ctx},
//		m.MockDB, nil,
//	)
//
//	<-m.Ctx.Done()
//}
