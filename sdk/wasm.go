package sdk

import "unsafe"

var activePlugin *Plugin
var wasmBuffer [64 * 1024]byte

// Run starts the SDK-managed WASM ABI for a plugin.
// Plugin authors only need to call this from the main package.
func Run(plugin *Plugin) {
	activePlugin = plugin
}

//go:wasmexport plugin_alloc
func wasmAlloc(size uint32) uint32 {
	if size > uint32(len(wasmBuffer)) {
		return 0
	}
	return uint32(uintptr(unsafe.Pointer(&wasmBuffer[0])))
}

//go:wasmexport plugin_free
func wasmFree(uint32, uint32) {}

//go:wasmexport plugin_handle_message
func wasmHandleMessage(pointer, length uint32) uint64 {
	if activePlugin == nil || pointer == 0 || length > uint32(len(wasmBuffer)) {
		return 0
	}

	request := unsafe.Slice((*byte)(unsafe.Pointer(uintptr(pointer))), length)
	response, err := activePlugin.HandleMessage(request)
	if err != nil || len(response) > len(wasmBuffer) {
		return 0
	}
	copy(wasmBuffer[:], response)
	responsePointer := uint32(uintptr(unsafe.Pointer(&wasmBuffer[0])))
	return uint64(responsePointer)<<32 | uint64(len(response))
}
