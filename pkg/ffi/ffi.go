// Package ffi provides C FFI exports for all refyne functionality.
//
// Build with:
//
//	CGO_ENABLED=1 go build -buildmode=c-shared -o librefyne.so ./pkg/ffi/
//
// All inputs/outputs are C strings. Complex data is JSON-serialized.
// The RefyneResult type provides both data and error fields.
// Callers must free results with refyne_result_free.
package ffi

// #include "refyne.h"
import "C"
import (
	"unsafe"

	refynecleaner "github.com/jmylchreest/refyne/pkg/cleaner/refyne"
)

// === Cleaner ===

//export refyne_clean
func refyne_clean(html *C.char, preset *C.char, format *C.char) C.RefyneResult {
	goHTML := C.GoString(html)
	goPreset := C.GoString(preset)
	goFormat := C.GoString(format)

	var cfg *refynecleaner.Config
	switch goPreset {
	case "minimal":
		cfg = refynecleaner.PresetMinimal()
	case "aggressive":
		cfg = refynecleaner.PresetAggressive()
	default:
		cfg = refynecleaner.DefaultConfig()
	}

	switch goFormat {
	case "text":
		cfg.Output = refynecleaner.OutputText
	case "html":
		cfg.Output = refynecleaner.OutputHTML
	default:
		cfg.Output = refynecleaner.OutputMarkdown
	}

	cleaner := refynecleaner.New(cfg)
	result, err := cleaner.Clean(goHTML)
	if err != nil {
		return makeError(err.Error())
	}

	return makeResult(result)
}

// === Memory Management ===

//export refyne_result_free
func refyne_result_free(result C.RefyneResult) {
	if result.data != nil {
		C.free(unsafe.Pointer(result.data))
	}
	if result.error != nil {
		C.free(unsafe.Pointer(result.error))
	}
}

// helpers

func makeResult(data string) C.RefyneResult {
	cData := C.CString(data)
	return C.RefyneResult{
		data:  cData,
		len:   C.int(len(data)),
		error: nil,
	}
}

func makeError(msg string) C.RefyneResult {
	cErr := C.CString(msg)
	return C.RefyneResult{
		data:  nil,
		len:   0,
		error: cErr,
	}
}

// main is required for c-shared build mode but should not be called.
func main() {}
