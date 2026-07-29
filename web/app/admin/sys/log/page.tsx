"use client"

import { logListApi, Log } from "@/api/log"
import { Crud, SearchField } from "@/components/widgets/crud"
import { DataListColumn } from "@/components/widgets/data-list"
import { cn } from "@/lib/utils"

const searchFields: SearchField[] = [
  {
    name: "request_path",
    label: "请求路径",
    type: "input",
    placeholder: "请求路径",
  },
  {
    name: "request_ip",
    label: "请求IP",
    type: "input",
    placeholder: "请求IP",
  },
  {
    name: "request_method",
    label: "请求方法",
    type: "select",
    options: [
      { value: "GET", label: "GET" },
      { value: "POST", label: "POST" },
      { value: "PUT", label: "PUT" },
      { value: "DELETE", label: "DELETE" },
    ],
  },
]

const methodColorMap: Record<string, string> = {
  GET: "text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-950",
  POST: "text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-950",
  PUT: "text-orange-600 dark:text-orange-400 bg-orange-50 dark:bg-orange-950",
  DELETE: "text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-950",
}

const methodColor = (method: string) =>
  methodColorMap[method] || "text-muted-foreground bg-muted"

const columns: DataListColumn<Log>[] = [
  {
    key: "id",
    header: "ID",
    cellClassName: "w-16 font-medium",
    cell: (row) => row.id,
  },
  {
    key: "account_name",
    header: "账号",
    cell: (row) => row.account_name || "-",
  },
  {
    key: "request_method",
    header: "方法",
    cell: (row) => (
      <span
        className={cn(
          "inline-flex items-center rounded px-2 py-0.5 text-xs font-medium",
          methodColor(row.request_method)
        )}
      >
        {row.request_method}
      </span>
    ),
  },
  {
    key: "request_path",
    header: "请求路径",
    cell: (row) => row.request_path || "-",
  },
  {
    key: "request_ip",
    header: "IP地址",
    cell: (row) => row.request_ip || "-",
  },
  {
    key: "response_code",
    header: "状态码",
    cell: (row) => (
      <span
        className={cn(
          row.response_code >= 200 && row.response_code < 300
            ? "text-green-600 dark:text-green-400"
            : "text-red-600 dark:text-red-400"
        )}
      >
        {row.response_code}
      </span>
    ),
  },
  {
    key: "process_time",
    header: "耗时",
    cell: (row) => row.process_time || "-",
  },
  {
    key: "description",
    header: "描述",
    cell: (row) => row.description || "-",
  },
]

export default function LogPage() {
  return (
    <Crud<Log>
      title="日志管理"
      entityName="日志"
      columns={columns}
      listApi={logListApi}
      pageSize={8}
      searchFields={searchFields}
    />
  )
}
