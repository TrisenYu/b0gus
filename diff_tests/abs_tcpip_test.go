package diff_tests

import (
	"b0gus/net_aux"
	"context"
	"math/rand/v2"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"b0gus/configs"
	"b0gus/internal/mock"
)

func TestNet(t *testing.T) {
	mockConf := atomic.Pointer[configs.LocalConfig]{}

	ctx, cancel := context.WithCancel(context.Background())
	obj := net_aux.ReentrantNetType{}
	restore := zap.ReplaceGlobals(zap.NewNop())
	defer restore()
	go func() {
		obj.Init(configs.RawEnum, nil) // temporarily set callback to nil
		obj.AlterNetFd(net_aux.TCPEnum, &mockConf)
		obj.EventMonitor(&mockConf, &configs.ServConcurrentCtrl{TerminatedCtx: ctx})
	}()
	cancel()
}

func TestMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	absServ := mock.NewMockAbsServNetAux(ctrl)
	configs.SetLogger(zap.NewNop())
	// 1. test with quit signal
	syncCh := make(chan struct{})
	for i := configs.RawEnum; i < configs.ENDofEnum; i++ {
		for j := net_aux.TCPEnum; j <= net_aux.OtherEnum; j++ {
			ctx, cancel := context.WithTimeout(context.TODO(), time.Duration(rand.Int64N(6))*time.Second)
			var (
				tmp    configs.LocalConfig
				ptrTmp atomic.Pointer[configs.LocalConfig]
				obj    net_aux.ReentrantNetType
			)
			go func() {
				err := gofakeit.Struct(&tmp)
				assert.NoError(t, err)
				ptrTmp.Store(&tmp)
				syncCh <- struct{}{}
			}()
			go func(x configs.ServEnum, y net_aux.NetTypeEnum) {
				<-syncCh
				obj.Init(x, absServ)
				obj.AlterNetFd(y, &ptrTmp)
				obj.EventMonitor(&ptrTmp, &configs.ServConcurrentCtrl{TerminatedCtx: ctx})
			}(i, j)
			// TODO: requirements of ports and filepath...
			select {
			case <-time.After(time.Duration(rand.Int64N(4)) * time.Second):
				cancel()
			case <-ctx.Done():
				cancel()
			}
		}
	}
	// [TODO] 2. remote interaction
}
