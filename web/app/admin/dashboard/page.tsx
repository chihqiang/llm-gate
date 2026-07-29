"use client"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { cn } from "@/lib/utils"

function StatCard({
  title,
  value,
  trend,
  trendLabel,
  valueClassName = "",
}: {
  title: string
  value: string
  trend?: string
  trendLabel?: string
  valueClassName?: string
}) {
  return (
    <Card>
      <CardHeader className="pb-4">
        <CardTitle className="text-sm font-medium text-muted-foreground">
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <div className={cn("text-3xl font-bold", valueClassName)}>{value}</div>
        {trend && trendLabel && (
          <div className="mt-1 text-sm text-green-600 dark:text-green-400">
            {trend} {trendLabel}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function ActivityItem({
  avatarLetter,
  avatarColor,
  title,
  time,
}: {
  avatarLetter: string
  avatarColor: "blue" | "green" | "purple"
  title: string
  time: string
}) {
  const colorClasses = {
    blue: "bg-blue-100 dark:bg-blue-900 text-blue-600 dark:text-blue-400",
    green: "bg-green-100 dark:bg-green-900 text-green-600 dark:text-green-400",
    purple:
      "bg-purple-100 dark:bg-purple-900 text-purple-600 dark:text-purple-400",
  }

  return (
    <div className="flex items-start space-x-3">
      <div
        className={cn(
          "flex h-8 w-8 items-center justify-center rounded-full",
          colorClasses[avatarColor]
        )}
      >
        <span className="font-medium">{avatarLetter}</span>
      </div>
      <div>
        <p className="text-sm font-medium">{title}</p>
        <p className="text-xs text-muted-foreground">{time}</p>
      </div>
    </div>
  )
}

function RecentActivity() {
  const activities = [
    {
      avatarLetter: "U",
      avatarColor: "blue" as const,
      title: "账户 John Doe 登录系统",
      time: "2分钟前",
    },
    {
      avatarLetter: "A",
      avatarColor: "green" as const,
      title: "新账户 Jane Smith 注册",
      time: "15分钟前",
    },
    {
      avatarLetter: "M",
      avatarColor: "purple" as const,
      title: "菜单管理已更新",
      time: "1小时前",
    },
  ]

  return (
    <Card>
      <CardHeader>
        <CardTitle>最近活动</CardTitle>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          {activities.map((activity, index) => (
            <ActivityItem key={index} {...activity} />
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

export default function DashboardPage() {
  const stats = [
    {
      title: "总账户数",
      value: "1,234",
      trend: "+12.5%",
      trendLabel: "较上月",
    },
    { title: "今日访问量", value: "567", trend: "+8.2%", trendLabel: "较昨日" },
    { title: "活跃账户", value: "892", trend: "+5.3%", trendLabel: "较上周" },
    {
      title: "系统状态",
      value: "正常",
      valueClassName: "text-green-600 dark:text-green-400",
      trendLabel: "所有服务运行正常",
    },
  ]

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-4">
        {stats.map((stat, index) => (
          <StatCard key={index} {...stat} />
        ))}
      </div>

      <RecentActivity />
    </div>
  )
}
