package plugin

import (
	"context"

	"github.com/tetratelabs/wazero"
)

var WasmContext context.Context
var Runtime wazero.Runtime

func Init() {
	WasmContext = context.Background()
	Runtime = wazero.NewRuntime(WasmContext)
}
