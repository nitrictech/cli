package eventbus

import "github.com/asaskevich/EventBus"

var bus EventBus.Bus

func Bus() EventBus.Bus {
	if bus == nil {
		bus = EventBus.New()
	}
	return bus
}
