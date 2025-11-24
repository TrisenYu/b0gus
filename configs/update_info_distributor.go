package configs

import "sync/atomic"

type ConfigMaintainer struct {
	updateCallback []func()
	blockedSign    chan struct{}
	initiated      atomic.Bool
}

func (cm *ConfigMaintainer) Init() {
	if cm.initiated.Load() {
		return
	}
	cm.initiated.Store(true)
	cm.blockedSign = make(chan struct{}, 1)
	cm.updateCallback = make([]func(), 0)
}

// Currently it seems that we don't have to unregister callback functions
func (cm *ConfigMaintainer) Regist(f func()) {
	if !cm.initiated.Load() {
		return
	}
	cm.blockedSign <- struct{}{}
	cm.updateCallback = append(cm.updateCallback, f)
	<-cm.blockedSign
}

// TODO: Fix this otherwise, developers may forget to close the channel
var UpdateFlag chan struct{} = make(chan struct{})

func (cm *ConfigMaintainer) UpdateConfig() {
	if !cm.initiated.Load() {
		return
	}
	for _, fn := range cm.updateCallback {
		// run each callback function in different go routine
		if fn == nil {
			continue
		}
		go fn()
	}
}

func (cm *ConfigMaintainer) SelfDestroy() {
	cm.initiated.Store(false)
	close(cm.blockedSign)
	cm.updateCallback = nil
}

var GlobConfigMaintainer ConfigMaintainer
