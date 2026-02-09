package ffi

// #include "refyne.h"
import "C"
import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/jmylchreest/refyne/pkg/model/inference"
)

// handleManager manages model inference handles with thread safety.
var handleManager = &modelHandles{
	models: make(map[C.int]inference.Inferencer),
}

type modelHandles struct {
	mu      sync.RWMutex
	models  map[C.int]inference.Inferencer
	nextID  C.int
}

func (h *modelHandles) add(inf inference.Inferencer) C.int {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.nextID++
	h.models[h.nextID] = inf
	return h.nextID
}

func (h *modelHandles) get(id C.int) (inference.Inferencer, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	inf, ok := h.models[id]
	return inf, ok
}

func (h *modelHandles) remove(id C.int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if inf, ok := h.models[id]; ok {
		inf.Close()
		delete(h.models, id)
	}
}

// === Model Inference ===

// modelLoadConfig is the JSON configuration for refyne_model_load.
type modelLoadConfig struct {
	NGPULayers int `json:"n_gpu_layers"`
	NCtx       int `json:"n_ctx"`
	NThreads   int `json:"n_threads"`
}

//export refyne_model_load
func refyne_model_load(modelPath *C.char, configJSON *C.char) C.int {
	goPath := C.GoString(modelPath)

	var opts []inference.LocalOption

	if configJSON != nil {
		goConfig := C.GoString(configJSON)
		var cfg modelLoadConfig
		if err := json.Unmarshal([]byte(goConfig), &cfg); err == nil {
			if cfg.NGPULayers > 0 {
				opts = append(opts, inference.WithGPULayers(cfg.NGPULayers))
			}
			if cfg.NCtx > 0 {
				opts = append(opts, inference.WithContextSize(cfg.NCtx))
			}
			if cfg.NThreads > 0 {
				opts = append(opts, inference.WithThreads(cfg.NThreads))
			}
		}
	}

	inf, err := inference.NewLocal(goPath, opts...)
	if err != nil {
		return -1
	}

	return handleManager.add(inf)
}

// modelInferOptions is the JSON configuration for per-request options.
type modelInferOptions struct {
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
	Grammar     string  `json:"grammar"`
}

//export refyne_model_infer
func refyne_model_infer(handle C.int, content *C.char, schemaJSON *C.char, optionsJSON *C.char) C.RefyneResult {
	inf, ok := handleManager.get(handle)
	if !ok {
		return makeError(fmt.Sprintf("invalid model handle: %d", handle))
	}

	goContent := C.GoString(content)
	goSchema := C.GoString(schemaJSON)

	// Build request
	req := inference.Request{
		Messages: []inference.Message{
			{Role: inference.RoleSystem, Content: "Extract structured data from the text by filling in the provided JSON template. Return only valid JSON matching the template structure."},
			{Role: inference.RoleUser, Content: fmt.Sprintf("### Template:\n%s\n### Text:\n%s", goSchema, goContent)},
		},
		MaxTokens:   2048,
		Temperature: 0.1,
	}

	// Apply options
	if optionsJSON != nil {
		goOpts := C.GoString(optionsJSON)
		var opts modelInferOptions
		if err := json.Unmarshal([]byte(goOpts), &opts); err == nil {
			if opts.MaxTokens > 0 {
				req.MaxTokens = opts.MaxTokens
			}
			if opts.Temperature > 0 {
				req.Temperature = opts.Temperature
			}
			if opts.Grammar != "" {
				req.Grammar = opts.Grammar
			}
		}
	}

	resp, err := inf.Infer(context.Background(), req)
	if err != nil {
		return makeError(err.Error())
	}

	return makeResult(resp.Content)
}

//export refyne_model_unload
func refyne_model_unload(handle C.int) {
	handleManager.remove(handle)
}
