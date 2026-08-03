"use client"

import { useState } from "react"
import { Account, AccountCreateUpdate } from "@/api/account"
import {
  accountListApi,
  accountCreateApi,
  accountUpdateApi,
  accountDeleteApi,
  accountDetailApi,
} from "@/api/account"
import { balanceAdjustApi } from "@/api/billing"
import { AccountForm } from "@/components/forms/account-form"
import { Crud, SearchField } from "@/components/widgets/crud"
import { DataListColumn } from "@/components/widgets/data-list"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
  DialogDescription,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { toast } from "sonner"
import { Wallet } from "lucide-react"
import { formatMoney } from "@/lib/utils"
import { hasMenuApiPermission } from "@/lib/account"

const searchFields: SearchField[] = [
  {
    name: "id",
    label: "账号ID",
    type: "input",
    placeholder: "搜索账号ID",
  },
]

const columns: DataListColumn<Account>[] = [
  {
    key: "id",
    header: "ID",
    cellClassName: "w-16 font-medium",
    cell: (row) => row.id,
  },
  {
    key: "name",
    header: "姓名",
    cell: (row) => row.name,
  },
  {
    key: "email",
    header: "邮箱",
    cell: (row) => row.email,
  },
  {
    key: "roles",
    header: "角色",
    cell: (row) => (row.roles || []).map((r) => r.name).join(", "),
  },
  {
    key: "balance_cents",
    header: "余额",
    cell: (row) => (
      <span className="font-medium">
        {formatMoney(row.balance_cents ?? 0)}
      </span>
    ),
  },
  {
    key: "status",
    header: "状态",
    cell: (row) => (row.status ? "活跃" : "禁用"),
  },
]

const defaultFormData: AccountCreateUpdate = {
  id: 0,
  name: "",
  email: "",
  password: "",
  roles: [],
  status: true,
  balance_cents: 0,
}

export default function AccountPage() {
  const [adjustAccount, setAdjustAccount] = useState<Account | null>(null)
  const [adjustAmount, setAdjustAmount] = useState("")
  const [adjustRemark, setAdjustRemark] = useState("")
  const [adjusting, setAdjusting] = useState(false)

  const canAdjust = hasMenuApiPermission("POST", "/api/v1/billing/balance/adjust")

  const openAdjust = (account: Account) => {
    setAdjustAccount(account)
    setAdjustAmount("")
    setAdjustRemark("")
  }

  const handleAdjust = async () => {
    if (!adjustAccount) return
    const amountCents = Math.round(parseFloat(adjustAmount) * 100)
    if (!adjustAmount || isNaN(amountCents) || amountCents === 0) {
      toast.error("请输入调整金额（正数加、负数减）")
      return
    }
    setAdjusting(true)
    try {
      await balanceAdjustApi({
        account_id: adjustAccount.id,
        amount_cents: amountCents,
        remark: adjustRemark.trim(),
      })
      toast.success("余额调整成功")
      setAdjustAccount(null)
      window.location.reload()
    } catch (error) {
      const msg = error instanceof Error ? error.message : "调整失败"
      toast.error(msg)
    } finally {
      setAdjusting(false)
    }
  }

  return (
    <>
      <Crud<Account, AccountCreateUpdate>
        title="账号管理"
        entityName="账号"
        columns={columns}
        listApi={accountListApi}
        pageSize={8}
        selectable
        searchFields={searchFields}
        deleteApi={accountDeleteApi}
        batchDelete
        createPermission="/api/v1/sys/accounts"
        updatePermission="/api/v1/sys/accounts/*"
        deletePermission="/api/v1/sys/accounts/*"
        formComponent={AccountForm}
        defaultFormData={defaultFormData}
        createApi={accountCreateApi}
        updateApi={accountUpdateApi}
        detailApi={accountDetailApi}
        extraActions={(account) =>
          canAdjust ? (
            <Button
              variant="ghost"
              size="icon-sm"
              className="text-muted-foreground hover:text-foreground"
              title="余额调整"
              onClick={(e) => {
                e.stopPropagation()
                openAdjust(account)
              }}
            >
              <Wallet className="size-3.5" />
            </Button>
          ) : null
        }
      />

      <Dialog
        open={!!adjustAccount}
        onOpenChange={(open) => !open && setAdjustAccount(null)}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>余额调整</DialogTitle>
            <DialogDescription>
              {adjustAccount
                ? `调整账号「${adjustAccount.name}」的余额，当前余额 ${formatMoney(
                    adjustAccount.balance_cents ?? 0
                  )}。输入正数增加、负数减少。`
                : ""}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-2">
            <div className="space-y-2">
              <Label htmlFor="adjust-amount">调整金额（元，可为负）</Label>
              <Input
                id="adjust-amount"
                type="number"
                step="0.01"
                placeholder="例如 50 或 -50"
                value={adjustAmount}
                onChange={(e) => setAdjustAmount(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="adjust-remark">备注</Label>
              <Input
                id="adjust-remark"
                placeholder="调整原因"
                value={adjustRemark}
                onChange={(e) => setAdjustRemark(e.target.value)}
              />
            </div>
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setAdjustAccount(null)}
              disabled={adjusting}
            >
              取消
            </Button>
            <Button onClick={handleAdjust} disabled={adjusting}>
              {adjusting && (
                <div className="mr-2 h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
              )}
              确认调整
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
