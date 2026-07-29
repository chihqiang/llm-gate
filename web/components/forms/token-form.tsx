"use client"

import { useEffect, useState } from "react"
import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import { LCheckbox, LSelect, LSelectOption } from "@/components/widgets/form-fields"
import { accountListApi } from "@/api/account"
import type { UserToken } from "@/api/llm"

interface TokenFormProps {
  formData: UserToken
  onChange: (data: UserToken) => void
}

export function TokenForm({ formData, onChange }: TokenFormProps) {
  const [accountOptions, setAccountOptions] = useState<LSelectOption[]>([])

  useEffect(() => {
    let cancelled = false
    accountListApi({ page: 1, size: 200 })
      .then((res) => {
        if (!cancelled) {
          setAccountOptions(
            res.data.map((a) => ({
              value: a.id,
              label: `${a.name} (${a.email})`,
            }))
          )
        }
      })
      .catch(() => {})
    return () => {
      cancelled = true
    }
  }, [])

  return (
    <div className="space-y-4 py-4">
      <div className="space-y-2">
        <LSelect
          id="account_id"
          label="所属账号"
          value={formData.account_id}
          options={accountOptions}
          onChange={(value) =>
            onChange({ ...formData, account_id: Number(value) })
          }
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="name">名称</Label>
        <Input
          id="name"
          value={formData.name}
          onChange={(e) => onChange({ ...formData, name: e.target.value })}
          placeholder="例如: 我的API Key"
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="quota">配额</Label>
        <Input
          id="quota"
          type="number"
          value={formData.quota}
          onChange={(e) =>
            onChange({ ...formData, quota: parseInt(e.target.value) || 0 })
          }
        />
      </div>
      <div className="flex items-center space-x-2">
        <LCheckbox
          id="status"
          label="启用状态"
          checked={formData.status}
          onChange={(status) => onChange({ ...formData, status })}
          className="mt-4"
        />
      </div>
    </div>
  )
}
