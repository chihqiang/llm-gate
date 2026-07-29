"use client"

import { useEffect, useState } from "react"
import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { LCheckbox, LSelect, LSelectOption } from "@/components/widgets/form-fields"
import { providerAllListApi } from "@/api/llm"
import type { ModelConfig } from "@/api/llm"

interface ModelFormProps {
  formData: ModelConfig
  onChange: (data: ModelConfig) => void
}

export function ModelForm({ formData, onChange }: ModelFormProps) {
  const [providerOptions, setProviderOptions] = useState<LSelectOption[]>([])

  useEffect(() => {
    let cancelled = false
    providerAllListApi()
      .then((providers) => {
        if (!cancelled) {
          setProviderOptions(
            providers.map((p) => ({ value: p.id, label: p.name }))
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
        <Label htmlFor="name">模型名称</Label>
        <Input
          id="name"
          value={formData.name}
          onChange={(e) => onChange({ ...formData, name: e.target.value })}
          placeholder="例如: gpt-4o"
        />
      </div>
      <div className="space-y-2">
        <LSelect
          id="provider_id"
          label="服务商"
          value={formData.provider_id}
          options={providerOptions}
          onChange={(value) =>
            onChange({ ...formData, provider_id: Number(value) })
          }
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="upstream_model_name">上游模型名</Label>
        <Input
          id="upstream_model_name"
          value={formData.upstream_model_name}
          onChange={(e) =>
            onChange({ ...formData, upstream_model_name: e.target.value })
          }
          placeholder="例如: gpt-4o-2024-08-06"
        />
      </div>
      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label htmlFor="model_ratio">模型倍率</Label>
          <Input
            id="model_ratio"
            type="number"
            step="0.01"
            value={formData.model_ratio}
            onChange={(e) =>
              onChange({
                ...formData,
                model_ratio: parseFloat(e.target.value) || 0,
              })
            }
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="completion_ratio">补全倍率</Label>
          <Input
            id="completion_ratio"
            type="number"
            step="0.01"
            value={formData.completion_ratio}
            onChange={(e) =>
              onChange({
                ...formData,
                completion_ratio: parseFloat(e.target.value) || 0,
              })
            }
          />
        </div>
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
      <div className="space-y-2">
        <Label htmlFor="remark">备注</Label>
        <Textarea
          id="remark"
          value={formData.remark}
          onChange={(e) => onChange({ ...formData, remark: e.target.value })}
          placeholder="备注信息"
        />
      </div>
    </div>
  )
}
