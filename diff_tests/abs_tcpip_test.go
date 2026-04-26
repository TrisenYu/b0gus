package diff_tests

import (
	"b0gus/configs"
	"b0gus/services"
	"context"
	"sync/atomic"
	"testing"
)

func TestNet(t *testing.T) {
	mockConf := atomic.Pointer[configs.LocalConfig]{}

	ctx, cancel := context.WithCancel(context.Background())
	obj := services.ReentrantNetType{}
	go func() {
		obj.Init(configs.RawEnum, nil) // temporarily set callback to nil
		obj.AlterNetFd(services.TCPEnum, &mockConf)
		obj.EventMonitor(&mockConf, &configs.ServConcurrentCtrl{Ctx: ctx})
	} ()

	cancel()
}
