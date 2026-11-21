"use client"

import { useCallback, useEffect, useState } from "react"
import {
  RechargeOrder,
  RechargeOrderStatus,
  orderListApi,
  orderCreateApi,
  orderConfirmApi,
  orderCancelApi,
} from "@/api/billing"
import { accountListApi } from "@/api/account"
import { DataList, DataListColumn } from "@/components/widgets/data-list"
import { LSelect, LSelectOption } from "@/components/widgets/form-fields"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from "@/components/ui/dialog"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import { Card } from "@/components/ui/card"
import { toast } from "sonner"
import { Plus } from "lucide-react"
import { formatMoney } from "@/lib/utils"
import { hasMenuApiPermission } from "@/lib/account"

const statusLabelMap: Record<RechargeOrderStatus, string> = {
  pending: "待确认",
  paid: "已入账",
  cancelled: "已取消",
}

const statusVariantMap: Record<
  RechargeOrderStatus,
  "default" | "secondary" | "outline"
> = {
  pending: "default",
  paid: "secondary",
  cancelled: "outline",
}

const columns: DataListColumn<RechargeOrder>[] = [
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
    key: "amount_cents",
    header: "金额",
    cell: (row) => <span className="font-medium">{formatMoney(row.amount_cents)}</span>,
  },
  {
    key: "status",
    header: "状态",
    cell: (row) => (
      <Badge variant={statusVariantMap[row.status]}>
        {statusLabelMap[row.status]}
      </Badge>
    ),
  },
  {
    key: "remark",
    header: "备注",
    cell: (row) => row.remark || "-",
  },
  {
    key: "paid_at",
    header: "入账时间",
    cell: (row) => row.paid_at || "-",
  },
  {
    key: "created_at",
    header: "创建时间",
    cell: (row) => row.created_at || "-",
  },
]

const pageSize = 10

export default function BillingOrdersPage() {
  const [orders, setOrders] = useState<RechargeOrder[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [status, setStatus] = useState<RechargeOrderStatus | "">("")

  const [createOpen, setCreateOpen] = useState(false)
  const [creating, setCreating] = useState(false)
  const [accountId, setAccountId] = useState<number>(0)
  const [amountYuan, setAmountYuan] = useState("")
  const [remark, setRemark] = useState("")
  const [accountOptions, setAccountOptions] = useState<LSelectOption[]>([])

  const [actingId, setActingId] = useState<number | null>(null)

  const fetchOrders = useCallback(async () => {
    setLoading(true)
    try {
      const res = await orderListApi({
        page,
        size: pageSize,
        status: status || undefined,
      })
      setOrders(res.data)
      setTotal(res.total)
    } catch (error) {
      const msg = error instanceof Error ? error.message : "获取充值订单失败"
      toast.error(msg)
    } finally {
      setLoading(false)
    }
  }, [page, status])

  useEffect(() => {
    fetchOrders()
  }, [fetchOrders])

  useEffect(() => {
    accountListApi({ page: 1, size: 200 })
      .then((res) => {
        setAccountOptions(
          res.data.map((a) => ({
            value: a.id,
            label: `${a.name} (${a.email})`,
          }))
        )
      })
      .catch(() => {})
  }, [])

  const handleCreate = async () => {
    const amountCents = Math.round(parseFloat(amountYuan) * 100)
    if (!accountId) {
      toast.error("请选择账号")
      return
    }
    if (!amountYuan || isNaN(amountCents) || amountCents <= 0) {
      toast.error("请输入正确的充值金额")
      return
    }
    setCreating(true)
    try {
      await orderCreateApi({
        account_id: accountId,
        amount_cents: amountCents,
        remark: remark.trim(),
      })
      toast.success("充值订单创建成功")
      setCreateOpen(false)
      setAccountId(0)
      setAmountYuan("")
      setRemark("")
      fetchOrders()
    } catch (error) {
      const msg = error instanceof Error ? error.message : "创建失败"
      toast.error(msg)
    } finally {
      setCreating(false)
    }
  }

  const handleConfirm = async (id: number) => {
    setActingId(id)
    try {
      await orderConfirmApi(id)
      toast.success("已确认入账")
      fetchOrders()
    } catch (error) {
      const msg = error instanceof Error ? error.message : "确认失败"
      toast.error(msg)
    } finally {
      setActingId(null)
    }
  }

  const handleCancel = async (id: number) => {
    setActingId(id)
    try {
      await orderCancelApi(id)
      toast.success("已取消")
      fetchOrders()
    } catch (error) {
      const msg = error instanceof Error ? error.message : "取消失败"
      toast.error(msg)
    } finally {
      setActingId(null)
    }
  }

  const canCreate = hasMenuApiPermission("POST", "/api/v1/billing/orders")
  const canConfirm = hasMenuApiPermission(
    "POST",
    "/api/v1/billing/orders/*/confirm"
  )
  const canCancel = hasMenuApiPermission(
    "POST",
    "/api/v1/billing/orders/*/cancel"
  )

  const finalColumns: DataListColumn<RechargeOrder>[] = [
    ...columns,
    {
      key: "actions",
      header: "操作",
      headerClassName: "text-right",
      cellClassName: "text-right",
      cell: (row) => (
        <div className="flex justify-end gap-1">
          {row.status === "pending" && canConfirm && (
            <Button
              variant="outline"
              size="sm"
              disabled={actingId === row.id}
              onClick={() => handleConfirm(row.id)}
            >
              确认入账
            </Button>
          )}
          {row.status === "pending" && canCancel && (
            <Button
              variant="ghost"
              size="sm"
              className="text-muted-foreground hover:text-destructive"
              disabled={actingId === row.id}
              onClick={() => handleCancel(row.id)}
            >
              取消
            </Button>
          )}
        </div>
      ),
    },
  ]

  return (
    <div className="space-y-4">
      <Card className="overflow-hidden">
        <div className="flex flex-wrap items-center justify-between gap-2 border-b bg-muted/20 px-4 py-3">
          <div className="flex items-center gap-2">
            <span className="text-sm whitespace-nowrap text-muted-foreground">
              状态
            </span>
            <Select
              value={status}
              onValueChange={(v) => setStatus((v || "") as RechargeOrderStatus | "")}
            >
              <SelectTrigger size="sm" className="w-28">
                <SelectValue placeholder="全部" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="">全部</SelectItem>
                <SelectItem value="pending">待确认</SelectItem>
                <SelectItem value="paid">已入账</SelectItem>
                <SelectItem value="cancelled">已取消</SelectItem>
              </SelectContent>
            </Select>
          </div>
          {canCreate && (
            <Button size="sm" onClick={() => setCreateOpen(true)}>
              <Plus className="mr-1 size-4" />
              新增充值订单
            </Button>
          )}
        </div>

        <DataList
          data={orders}
          columns={finalColumns}
          keyExtractor={(row) => String(row.id)}
          loading={loading}
          pagination={{ page, pageSize, total, onPageChange: setPage }}
          emptyText="暂无充值订单"
          bordered={false}
        />
      </Card>

      <Dialog open={createOpen} onOpenChange={(open) => !open && setCreateOpen(false)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>新增充值订单</DialogTitle>
            <DialogDescription>
              线下转账后，管理员确认订单即可入账到对应账户。
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-2">
            <LSelect
              id="order-account"
              label="充值账号"
              value={accountId || undefined}
              options={accountOptions}
              onChange={(value) => setAccountId(Number(value) || 0)}
            />
            <div className="space-y-2">
              <Label htmlFor="order-amount">充值金额（元）</Label>
              <Input
                id="order-amount"
                type="number"
                step="0.01"
                min="0"
                placeholder="例如 100.00"
                value={amountYuan}
                onChange={(e) => setAmountYuan(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="order-remark">备注</Label>
              <Input
                id="order-remark"
                placeholder="例如：线下转账 100 元"
                value={remark}
                onChange={(e) => setRemark(e.target.value)}
              />
            </div>
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setCreateOpen(false)}
              disabled={creating}
            >
              取消
            </Button>
            <Button onClick={handleCreate} disabled={creating}>
              {creating && (
                <div className="mr-2 h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
              )}
              创建
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
