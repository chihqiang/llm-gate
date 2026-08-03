"use client"

import {
  BalanceTransaction,
  TransactionType,
  transactionListApi,
} from "@/api/billing"
import { Crud, SearchField } from "@/components/widgets/crud"
import { DataListColumn } from "@/components/widgets/data-list"
import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"
import { formatSignedMoney, formatMoney } from "@/lib/utils"

const typeLabelMap: Record<TransactionType, string> = {
  consume: "消费",
  refund: "退款",
  recharge: "充值",
  adjust: "调整",
}

const typeClassMap: Record<TransactionType, string> = {
  consume: "text-red-600 dark:text-red-400",
  refund: "text-green-600 dark:text-green-400",
  recharge: "text-green-600 dark:text-green-400",
  adjust: "text-blue-600 dark:text-blue-400",
}

const searchFields: SearchField[] = [
  {
    name: "account_id",
    label: "账号ID",
    type: "input",
    placeholder: "账号ID",
  },
  {
    name: "type",
    label: "类型",
    type: "select",
    options: [
      { value: "consume", label: "消费" },
      { value: "refund", label: "退款" },
      { value: "recharge", label: "充值" },
      { value: "adjust", label: "调整" },
    ],
  },
]

const columns: DataListColumn<BalanceTransaction>[] = [
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
    key: "type",
    header: "类型",
    cell: (row) => (
      <Badge
        variant="secondary"
        className={cn(typeClassMap[row.type], "border-transparent")}
      >
        {typeLabelMap[row.type]}
      </Badge>
    ),
  },
  {
    key: "amount_cents",
    header: "变动金额",
    cell: (row) => (
      <span
        className={cn(
          "font-medium",
          row.amount_cents < 0
            ? "text-red-600 dark:text-red-400"
            : "text-green-600 dark:text-green-400"
        )}
      >
        {formatSignedMoney(row.amount_cents)}
      </span>
    ),
  },
  {
    key: "balance_cents",
    header: "变动后余额",
    cell: (row) => formatMoney(row.balance_cents),
  },
  {
    key: "token_id",
    header: "TokenID",
    cell: (row) => (row.token_id ? row.token_id : "-"),
  },
  {
    key: "remark",
    header: "备注",
    cell: (row) => row.remark || "-",
  },
  {
    key: "created_at",
    header: "时间",
    cell: (row) => row.created_at || "-",
  },
]

export default function BillingTransactionsPage() {
  return (
    <Crud<BalanceTransaction>
      title="资金流水"
      entityName="资金流水"
      columns={columns}
      listApi={transactionListApi}
      pageSize={10}
      searchFields={searchFields}
      dialogWidth="900px"
    />
  )
}
