package services

import (
	"b0gus/configs"
	"context"
	"strings"

	"github.com/emiago/sipgo"
	"github.com/emiago/sipgo/sip"
)

func sipAckHandler(req *sip.Request, tx sip.ServerTransaction) {
	var sb strings.Builder
	sb.WriteString(req.CallID().Name())
	sb.WriteRune(' ')
	sb.WriteString(req.Laddr.String())
	configs.Logger.Info(sb.String())
}

func sipByeHandler(req *sip.Request, tx sip.ServerTransaction) {
	res := sip.NewResponseFromRequest(req, 200, "OK", nil)
	_ = tx.Respond(res)
}

// example is the minimized PoC for setting up SIP message control server,
// which is temporarily not used.
//
//nolint:unused
func example() {
	defer configs.Logger.Info("sip services is closed")
	// setup service
	ua, err := sipgo.NewUA(
		sipgo.WithUserAgent("sip-server"),
	)
	if err != nil {
		configs.Logger.Error(err.Error())
		return
	}
	serv, err := sipgo.NewServer(ua)
	if err != nil {
		configs.Logger.Error(err.Error())
		return
	}
	defer func() {
		_ = serv.Close()
	}()
	serv.OnAck(sipAckHandler)
	serv.OnBye(sipByeHandler)
	serv.OnInvite(nil)
	serv.OnRegister(nil)
	serv.OnMessage(nil)
	serv.OnUpdate(nil)
	serv.OnInfo(nil)
	serv.OnNotify(nil)

	// listen up?
	err = serv.ListenAndServe(context.TODO(), "udp", ":5678")
	if err != nil {
		configs.Logger.Error(err.Error())
	}
}
