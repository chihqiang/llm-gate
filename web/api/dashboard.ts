import request from "@/lib/request"

export interface DashboardStats {
  total_requests: number
  today_requests: number
  total_tokens: number
  today_tokens: number
  total_quota: number
  active_tokens: number
  total_providers: number
  total_models: number
}

export async function dashboardStatsApi(): Promise<DashboardStats> {
  return await request.get<DashboardStats>("/api/v1/dashboard/stats")
}
