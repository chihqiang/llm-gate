package relay

import "chihqiang/llm-gate/model"

type ResolveResult struct {
	Model    *model.ModelConfig
	Provider *model.Provider
}
