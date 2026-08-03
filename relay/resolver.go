package relay

import "chihqiang/llm-gate/model"

// upstreamCandidate 一次转发尝试的候选（模型 + 服务商），用于多服务商降级。
type upstreamCandidate struct {
	Model    *model.ModelConfig
	Provider *model.Provider
}

// ResolveResult 模型解析结果。Candidates 按优先级排序，首个为加权随机选中的主候选。
type ResolveResult struct {
	Model      *model.ModelConfig
	Provider   *model.Provider
	Candidates []upstreamCandidate
}
