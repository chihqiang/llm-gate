import request from "@/lib/request"

export interface DashboardStats {
  total_accounts: number
  today_visits: number
  active_accounts: number
  system_status: string
}

export async function dashboardStatsApi(): Promise<DashboardStats> {
  return await request.get<DashboardStats>("/api/v1/dashboard/stats")
}
