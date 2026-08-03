import request, { PageRequest, PageResponse } from "@/lib/request"

// ==================== Provider ====================

export interface Provider {
  id: number
  name: string
  base_url: string
  api_key?: string
  status: boolean
  priority: number
  weight: number
  remark: string
  created_at: string
  updated_at: string
}

export interface ProviderListRequest extends PageRequest {
  name?: string
}

export interface ProviderListResponse extends PageResponse<Provider> {}

export async function providerListApi(
  data: ProviderListRequest
): Promise<ProviderListResponse> {
  return await request.get<ProviderListResponse>("/api/v1/llm/providers", {
    params: data,
  })
}

export async function providerAllListApi(): Promise<Provider[]> {
  return await request.get<Provider[]>("/api/v1/llm/providers/all")
}

export async function providerDetailApi(id: number): Promise<Provider> {
  return await request.get<Provider>(`/api/v1/llm/providers/${id}`)
}

export async function providerCreateApi(
  data: Partial<Provider>
): Promise<Provider> {
  return await request.post<Provider>("/api/v1/llm/providers", data)
}

export async function providerUpdateApi(
  data: Partial<Provider> & { id: number }
): Promise<Provider> {
  return await request.put<Provider>(`/api/v1/llm/providers/${data.id}`, data)
}

export async function providerDeleteApi(id: number): Promise<void> {
  return await request.delete(`/api/v1/llm/providers/${id}`)
}

export interface UpstreamModel {
  id: string
  exists: boolean
}

export interface SyncModelsPreview {
  total: number
  models: UpstreamModel[]
}

export async function providerPreviewSyncModelsApi(
  id: number
): Promise<SyncModelsPreview> {
  return await request.get<SyncModelsPreview>(
    `/api/v1/llm/providers/${id}/sync-models/preview`
  )
}

export interface SyncModelsResult {
  total: number
  created: number
  skipped: number
  models: string[]
}

export async function providerSyncModelsApi(
  id: number,
  models: string[]
): Promise<SyncModelsResult> {
  return await request.post<SyncModelsResult>(
    `/api/v1/llm/providers/${id}/sync-models`,
    { models }
  )
}

// ==================== Model Config ====================

export interface ModelConfig {
  id: number
  name: string
  provider_id: number
  upstream_model_name: string
  model_ratio: number
  completion_ratio: number
  weight: number
  status: boolean
  remark: string
  created_at: string
  updated_at: string
}

export interface ModelListRequest extends PageRequest {
  name?: string
  provider_id?: number
}

export interface ModelListResponse extends PageResponse<ModelConfig> {}

export async function modelListApi(
  data: ModelListRequest
): Promise<ModelListResponse> {
  return await request.get<ModelListResponse>("/api/v1/llm/models", {
    params: data,
  })
}

export async function modelAllListApi(): Promise<ModelConfig[]> {
  return await request.get<ModelConfig[]>("/api/v1/llm/models/all")
}

export async function modelDetailApi(id: number): Promise<ModelConfig> {
  return await request.get<ModelConfig>(`/api/v1/llm/models/${id}`)
}

export async function modelCreateApi(
  data: Partial<ModelConfig>
): Promise<ModelConfig> {
  return await request.post<ModelConfig>("/api/v1/llm/models", data)
}

export async function modelUpdateApi(
  data: Partial<ModelConfig> & { id: number }
): Promise<ModelConfig> {
  return await request.put<ModelConfig>(`/api/v1/llm/models/${data.id}`, data)
}

export async function modelDeleteApi(id: number): Promise<void> {
  return await request.delete(`/api/v1/llm/models/${id}`)
}

// ==================== User Token ====================

export interface UserToken {
  id: number
  account_id: number
  name: string
  key: string
  key_masked: string
  quota: number
  spent_cents: number
  model_ids: number[]
  status: boolean
  expired_at: string | null
  created_at: string
  updated_at: string
}

export interface TokenListRequest extends PageRequest {
  account_id?: number
}

export interface TokenListResponse extends PageResponse<UserToken> {}

export async function tokenListApi(
  data: TokenListRequest
): Promise<TokenListResponse> {
  return await request.get<TokenListResponse>("/api/v1/llm/tokens", {
    params: data,
  })
}

export async function tokenDetailApi(id: number): Promise<UserToken> {
  return await request.get<UserToken>(`/api/v1/llm/tokens/${id}`)
}

export async function tokenCreateApi(
  data: Partial<UserToken>
): Promise<UserToken> {
  return await request.post<UserToken>("/api/v1/llm/tokens", data)
}

export async function tokenUpdateApi(
  data: Partial<UserToken> & { id: number }
): Promise<UserToken> {
  return await request.put<UserToken>(`/api/v1/llm/tokens/${data.id}`, data)
}

export async function tokenDeleteApi(id: number): Promise<void> {
  return await request.delete(`/api/v1/llm/tokens/${id}`)
}

export async function tokenRevealApi(id: number): Promise<string> {
  const res = await request.get<{ key: string }>(
    `/api/v1/llm/tokens/${id}/reveal`
  )
  return res.key
}

// ==================== Usage Log ====================

export interface UsageLog {
  id: number
  account_id: number
  token_id: number
  model_name: string
  provider_id: number
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
  quota_cost: number
  cost_cents: number
  estimated: boolean
  request_id: string
  created_at: string
  account_name: string
  token_name: string
}

export interface UsageStat {
  model_name: string
  total_tokens: number
  total_quota_cost: number
  request_count: number
}

export interface UsageListRequest extends PageRequest {
  account_id?: number
  model_name?: string
  start_date?: string
  end_date?: string
}

export interface UsageListResponse extends PageResponse<UsageLog> {}

export async function usageListApi(
  data: UsageListRequest
): Promise<UsageListResponse> {
  return await request.get<UsageListResponse>("/api/v1/llm/usage", {
    params: data,
  })
}

export async function usageStatsApi(params: {
  account_id?: number
  start_date?: string
  end_date?: string
}): Promise<UsageStat[]> {
  return await request.get<UsageStat[]>("/api/v1/llm/usage/stats", {
    params,
  })
}
