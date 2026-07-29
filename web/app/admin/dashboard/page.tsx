"use client"

import { useState, useEffect, useCallback } from "react"
import {
  UsageLog,
  usageListApi,
  usageStatsApi,
  UsageStat,
} from "@/api/llm"
import {
  DashboardStats,
  dashboardStatsApi,
} from "@/api/dashboard"
import { Crud, SearchField } from "@/components/widgets/crud"
import { DataListColumn } from "@/components/widgets/data-list"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"

function StatCard({
  title,
  value,
  loading,
}: {
  title: string
  value: string
  loading?: boolean
}) {
  return (
    <Card>
      <CardHeader className="pb-4">
        <CardTitle className="text-sm font-medium text-muted-foreground">
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className="text-3xl font-bold">
          {loading ? "-" : value}
        </div>
      </CardContent>
    </Card>
  )
}

const usageSearchFields: SearchField[] = [
  {
    name: "model_name",
    label: "模型",
    type: "input",
    placeholder: "搜索模型",
  },
  {
    name: "start_date",
    label: "开始日期",
    type: "input",
    placeholder: "2024-01-01",
  },
  {
    name: "end_date",
    label: "结束日期",
    type: "input",
    placeholder: "2024-12-31",
  },
]

const usageColumns: DataListColumn<UsageLog>[] = [
  {
    key: "id",
    header: "ID",
    cellClassName: "w-16 font-medium",
    cell: (row) => row.id,
  },
  {
    key: "account_id",
    header: "账号ID",
    cell: (row) => row.account_id,
  },
  {
    key: "model_name",
    header: "模型",
    cell: (row) => row.model_name,
  },
  {
    key: "prompt_tokens",
    header: "输入Token",
    cell: (row) => row.prompt_tokens.toLocaleString(),
  },
  {
    key: "completion_tokens",
    header: "输出Token",
    cell: (row) => row.completion_tokens.toLocaleString(),
  },
  {
    key: "total_tokens",
    header: "总Token",
    cell: (row) => row.total_tokens.toLocaleString(),
  },
  {
    key: "quota_cost",
    header: "消耗配额",
    cell: (row) => row.quota_cost.toLocaleString(),
  },
  {
    key: "created_at",
    header: "时间",
    cell: (row) => row.created_at || "-",
  },
]

export default function DashboardPage() {
  const [dashStats, setDashStats] = useState<DashboardStats | null>(null)
  const [dashLoading, setDashLoading] = useState(true)
  const [usageStats, setUsageStats] = useState<UsageStat[]>([])
  const [usageLoading, setUsageLoading] = useState(true)

  const fetchData = useCallback(async () => {
    setDashLoading(true)
    setUsageLoading(true)
    try {
      const [ds, us] = await Promise.all([
        dashboardStatsApi(),
        usageStatsApi({}),
      ])
      setDashStats(ds)
      setUsageStats(us ?? [])
    } catch {
      // ignore
    } finally {
      setDashLoading(false)
      setUsageLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  const totalTokens = usageStats.reduce((sum, s) => sum + s.total_tokens, 0)
  const totalQuota = usageStats.reduce((sum, s) => sum + s.total_quota_cost, 0)
  const totalRequests = usageStats.reduce((sum, s) => sum + s.request_count, 0)

  const statCards = [
    { title: "总账户数", value: dashStats?.total_accounts.toLocaleString() ?? "-" },
    { title: "今日访问量", value: dashStats?.today_visits.toLocaleString() ?? "-" },
    { title: "活跃账户(7日)", value: dashStats?.active_accounts.toLocaleString() ?? "-" },
    { title: "系统状态", value: dashStats?.system_status ?? "-" },
  ]

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-4">
        {statCards.map((stat, index) => (
          <StatCard key={index} title={stat.title} value={stat.value} loading={dashLoading} />
        ))}
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm text-muted-foreground">
              总请求数
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {usageLoading ? "-" : totalRequests.toLocaleString()}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm text-muted-foreground">
              总 Token 消耗
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {usageLoading ? "-" : totalTokens.toLocaleString()}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm text-muted-foreground">
              总配额消耗
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {usageLoading ? "-" : totalQuota.toLocaleString()}
            </div>
          </CardContent>
        </Card>
      </div>

      <Crud<UsageLog>
        title="用量明细"
        entityName="用量记录"
        columns={usageColumns}
        listApi={usageListApi}
        pageSize={8}
        searchFields={usageSearchFields}
        dialogWidth="900px"
      />
    </div>
  )
}
