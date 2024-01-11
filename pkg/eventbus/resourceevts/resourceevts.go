package resourceevts

import (
	"net"

	"github.com/asaskevich/EventBus"
	"github.com/nitrictech/nitric/core/pkg/gateway"
)

type LocalInfrastructureState struct {
	GatewayStartOpts   *gateway.GatewayStartOpts
	TriggerAddress     string
	StorageAddress     string
	ApiAddresses       map[string]string
	WebSocketAddresses map[string]string
	ServiceListener    net.Listener
}

const localInfrastructureTopic = "local_infrastructure"

var localInfrastructureStateBus EventBus.Bus

func localInfrastructure() EventBus.Bus {
	if localInfrastructureStateBus == nil {
		localInfrastructureStateBus = EventBus.New()
	}

	return localInfrastructureStateBus
}

func Subscribe(fn func(opts *LocalInfrastructureState)) error {
	return localInfrastructure().Subscribe(localInfrastructureTopic, fn)
}

func SubscribeAsync(fn func(opts *LocalInfrastructureState)) error {
	return localInfrastructure().SubscribeAsync(localInfrastructureTopic, fn, false)
}

func SubscribeOnce(fn func(opts *LocalInfrastructureState)) error {
	return localInfrastructure().SubscribeOnce(localInfrastructureTopic, fn)
}

func NewSubChannel() chan LocalInfrastructureState {
	ch := make(chan LocalInfrastructureState)
	SubscribeAsync(func(opts *LocalInfrastructureState) {
		ch <- *opts
	})

	return ch
}

func Publish(evt LocalInfrastructureState) {
	localInfrastructure().Publish(localInfrastructureTopic, evt)
}
