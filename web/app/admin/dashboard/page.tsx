"use client"

import { useState, useEffect, useCallback } from "react"
import { UsageLog, usageListApi } from "@/api/llm"
import { DashboardStats, dashboardStatsApi } from "@/api/dashboard"
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
        <div className="text-3xl font-bold">{loading ? "-" : value}</div>
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
    header: "用户ID",
    cell: (row) => row.account_id,
  },
  {
    key: "token_name",
    header: "令牌",
    cell: (row) => row.token_name || "-",
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
  const fetchData = useCallback(async () => {
    setDashLoading(true)
    try {
      setDashStats(await dashboardStatsApi())
    } catch {
      // ignore
    } finally {
      setDashLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  const topCards = [
    {
      title: "总请求数",
      value: dashStats ? dashStats.total_requests.toLocaleString() : "-",
    },
    {
      title: "今日请求",
      value: dashStats ? dashStats.today_requests.toLocaleString() : "-",
    },
    {
      title: "总 Token",
      value: dashStats ? dashStats.total_tokens.toLocaleString() : "-",
    },
    {
      title: "今日 Token",
      value: dashStats ? dashStats.today_tokens.toLocaleString() : "-",
    },
    {
      title: "总配额消耗",
      value: dashStats ? dashStats.total_quota.toLocaleString() : "-",
    },
    {
      title: "活跃 Key",
      value: dashStats ? dashStats.active_tokens.toLocaleString() : "-",
    },
    {
      title: "服务商数",
      value: dashStats ? dashStats.total_providers.toLocaleString() : "-",
    },
    {
      title: "模型数",
      value: dashStats ? dashStats.total_models.toLocaleString() : "-",
    },
  ]

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4 xl:grid-cols-8">
        {topCards.map((stat, index) => (
          <StatCard
            key={index}
            title={stat.title}
            value={stat.value}
            loading={dashLoading}
          />
        ))}
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
