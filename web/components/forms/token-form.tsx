"use client"

import { useEffect, useState } from "react"
import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import {
  LCheckbox,
  LSelect,
  LSelectOption,
} from "@/components/widgets/form-fields"
import { accountListApi } from "@/api/account"
import { modelAllListApi, type UserToken } from "@/api/llm"

interface TokenFormProps {
  formData: UserToken
  onChange: (data: UserToken) => void
}

export function TokenForm({ formData, onChange }: TokenFormProps) {
  const [accountOptions, setAccountOptions] = useState<LSelectOption[]>([])
  const [modelOptions, setModelOptions] = useState<LSelectOption[]>([])

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
    modelAllListApi()
      .then((models) => {
        if (!cancelled) {
          setModelOptions(
            models.map((m) => ({
              value: m.id,
              label: `${m.name} (#${m.id})`,
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
        <Label htmlFor="quota">预算（元，0=不限）</Label>
        <Input
          id="quota"
          type="number"
          min="0"
          step="0.01"
          value={formData.quota > 0 ? (formData.quota / 100).toFixed(2) : "0"}
          onChange={(e) =>
            onChange({
              ...formData,
              quota: Math.round(parseFloat(e.target.value) * 100) || 0,
            })
          }
        />
        <p className="text-xs text-muted-foreground">
          该 Key 累计消费上限，超出后请求将被拒绝。单位：元
        </p>
      </div>
      <div className="space-y-2">
        <LSelect
          id="model_ids"
          label="模型白名单（不选=全部模型）"
          value={formData.model_ids}
          options={modelOptions}
          multiSelect
          onChange={(value) =>
            onChange({
              ...formData,
              model_ids: (value ?? []).map((v) => Number(v)),
            })
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
