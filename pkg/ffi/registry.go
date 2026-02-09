package ffi

// #include "refyne.h"
import "C"
import (
	"context"
	"encoding/json"

	"github.com/jmylchreest/refyne/pkg/model/registry"
)

//export refyne_registry_list
func refyne_registry_list(registryPath *C.char) C.RefyneResult {
	goPath := C.GoString(registryPath)

	reg, err := registry.NewFlatFile(goPath, "")
	if err != nil {
		return makeError(err.Error())
	}

	models, err := reg.List(context.Background())
	if err != nil {
		return makeError(err.Error())
	}

	data, err := json.Marshal(models)
	if err != nil {
		return makeError(err.Error())
	}

	return makeResult(string(data))
}

//export refyne_registry_ensure_cached
func refyne_registry_ensure_cached(registryPath *C.char, modelName *C.char, cacheDirPath *C.char) C.RefyneResult {
	goRegistryPath := C.GoString(registryPath)
	goModelName := C.GoString(modelName)

	cacheDir := ""
	if cacheDirPath != nil {
		cacheDir = C.GoString(cacheDirPath)
	}

	reg, err := registry.NewFlatFile(goRegistryPath, cacheDir)
	if err != nil {
		return makeError(err.Error())
	}

	localPath, err := reg.EnsureCached(context.Background(), goModelName)
	if err != nil {
		return makeError(err.Error())
	}

	return makeResult(localPath)
}
