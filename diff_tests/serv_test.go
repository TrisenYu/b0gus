package diff_tests

import (
	"b0gus/configs"
	"b0gus/crypto_aux"
	"b0gus/internal/mock"
	"b0gus/services"

	"context"
	"io"
	"math/rand/v2"
	"net"
	"os"
	"strconv"
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
	/* mock-related */
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
		m.conf.SSHconfig = p
	case configs.SMTPconfig:
		m.conf.SMTPconfig = p
	case configs.HTTPconfig:
		m.conf.HTTPconfig = p
	case configs.NTPconfig:
		m.conf.NTPconfig = p
	case configs.DNSconfig:
		m.conf.DNSconfig = p
	default:
		return
	}
	m.FakeConf.Store(&m.conf)
}

func (m *mockEvaluator) Init(t *testing.T, timeout time.Duration) {
	m.initCtrl(t)
	m.Ctx, m.Cancel = context.WithTimeout(context.Background(), timeout)
	m.FakeConf.Store(&m.conf)
	m.MockDB = mock.NewMockDBhandler(m.ctrl)
	m.MockDB.EXPECT().Setup(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
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
	val.ListenPort = 32222
	val.DBPath = "./test_dir/youme.db"
	m.ReloadByAssignment(val)
	restore := zap.ReplaceGlobals(zap.NewNop())
	defer restore()
	tmp, ok := m.FakeConf.Load().SelectTerm(configs.SSHEnum).(configs.SSHconfig)
	assert.True(t, ok, "Expected to find SSH config")
	if !ok {
		t.FailNow()
	}
	assert.True(t, configs.CheckSSHconfig(&tmp))
	quitSign := atomic.Bool{}
	quitSign.Store(false)
	go func() {
		(&services.SSHServConf{DbFd: m.MockDB}).Run(
			&m.FakeConf,
			&configs.ServConcurrentCtrl{TerminatedCtx: m.Ctx},
			servKey,
		)
		quitSign.Store(true)
	} ()
	sshConf := &ssh.ClientConfig{
		User: gofakeit.Username(),
		Auth: []ssh.AuthMethod{
			ssh.Password(
				gofakeit.Password(true, true, true, true, false, 7),
			),
		},
		HostKeyCallback: ssh.FixedHostKey(servKey.PublicKey()),
		Timeout:         time.Second * 10,
	}
	if quitSign.Load() {
		m.Cancel()
		t.Error("fake ssh server quit before dialing")
		return
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
	quitSign := atomic.Bool{}
	quitSign.Store(false)
	go func() {
		(&services.NTPServConf{DbFd: m.MockDB}).Run(
			&m.FakeConf,
			&configs.ServConcurrentCtrl{TerminatedCtx: m.Ctx},
			nil,
		)
		quitSign.Store(true)
	} ()

	if quitSign.Load() {
		m.Cancel()
		t.Error("quit before dialing")
		return
	}
	conn, err := net.Dial("udp", "localhost:"+strconv.Itoa(int(val.ListenPort)))
	assert.NoError(t, err)
	if conn == nil {
		m.Cancel()
		t.Fatal("unable to use UDP connection")
		return
	}
	defer func() { _ = conn.Close() }()
	for _, cmd := range CmdSeeds {
		/* only write UDP connection */
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
	quitSign := atomic.Bool{}
	quitSign.Store(false)
	go func() {
		(&services.DNSservConf{DBFd: m.MockDB}).Run(
			&m.FakeConf,
			&configs.ServConcurrentCtrl{TerminatedCtx: m.Ctx},
			nil,
		)
		quitSign.Store(true)
	} ()

	c := dns.Client{
		Net:     "udp",
		Timeout: 2 * time.Second,
	}
	for _, testcase := range dnsTestcase {
		dm := new(dns.Msg)
		dm.SetQuestion(dns.Fqdn(testcase.domain), testcase.dnsType)
		dm.RecursionDesired = true
		if quitSign.Load() {
			t.Error("exchange before quiting")
			m.Cancel()
			break
		}
		in, _, err := c.Exchange(dm, "localhost:"+strconv.Itoa(int(val.ListenPort)))
		assert.NoError(t, err)
		if err != nil {
			m.Cancel()
			t.Fatalf("unable to use UDP connection for DNS server. err: %v", err)
			return
		}
		if in.Rcode != dns.RcodeSuccess {
			m.Cancel()
			t.Fatalf("DNS Fault: %s", dns.RcodeToString[in.Rcode])
		}
	}
	<-m.Ctx.Done()
}

func TestMockBrancher(t *testing.T) {
	termCh := make(chan struct{})
	defer close(termCh)
	m := mockEvaluator{}
	m.Init(t, 15*time.Second)
	defer m.Free()

	m.ReloadConf(t)
	val, ok := m.Access(configs.DNSEnum).(configs.DNSconfig)
	assert.True(t, ok, "Expected to find DNS config")
	val.ListenPort = 5553
	val.DBPath = "./test_dir/hello.db"
	m.ReloadByAssignment(val)
	defer m.Cancel()
	restore := zap.ReplaceGlobals(zap.NewNop())
	defer restore()
	go services.Brancher(termCh, &m.FakeConf)
	<-time.After(15 * time.Second)
	termCh <- struct{}{}
}
