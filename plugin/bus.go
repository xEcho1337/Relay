package plugin

import "relay/sdk"

type Bus = sdk.Bus

func NewBus() *Bus {
	return sdk.NewBus()
}

var DefaultBus = sdk.DefaultBus
