package diff_tests

import (
	"b0gus/configs"
	"b0gus/internal/mock"
	"b0gus/services"
	"math/rand/v2"
	"time"

	"context"
	"sync/atomic"
	"testing"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestNet(t *testing.T) {
	mockConf := atomic.Pointer[configs.LocalConfig]{}

	ctx, cancel := context.WithCancel(context.Background())
	obj := services.ReentrantNetType{}
	go func() {
		obj.Init(configs.RawEnum, nil) // temporarily set callback to nil
		obj.AlterNetFd(services.TCPEnum, &mockConf)
		obj.EventMonitor(&mockConf, &configs.ServConcurrentCtrl{Ctx: ctx})
	}()

	cancel()
}

func TestMock(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	var (
		tmp    configs.LocalConfig
		ptrTmp atomic.Pointer[configs.LocalConfig]
		obj    services.ReentrantNetType
	)

	absServ := mock.NewMockAbsServNetAux(ctrl)
	// 1. quit signal
	for i := configs.RawEnum; i < configs.ENDofEnum; i++ {
		for j := services.TCPEnum; j <= services.OtherEnum; j++ {
			ctx, cancel := context.WithTimeout(context.TODO(), time.Duration(rand.Int64N(4))*time.Second)

			err := gofakeit.Struct(&tmp) // TODO: requirements of ports and filepath...
			assert.NoError(t, err)
			ptrTmp.Store(&tmp)

			go func() {
				obj.Init(i, absServ)
				obj.AlterNetFd(j, &ptrTmp)
				obj.EventMonitor(&ptrTmp, &configs.ServConcurrentCtrl{Ctx: ctx})
			}()
			select {
			case <-ctx.Done():
				cancel()
			default:
				time.Sleep(time.Duration(rand.Int64N(6)) * time.Second)
				cancel()
			}
		}
	}
	// 2. remote interaction
	// TODO: do not port...
}
