"use client"

import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { LCheckbox } from "@/components/widgets/form-fields"
import type { Provider } from "@/api/llm"

interface ProviderFormProps {
  formData: Provider
  onChange: (data: Provider) => void
}

export function ProviderForm({ formData, onChange }: ProviderFormProps) {
  return (
    <div className="space-y-4 py-4">
      <div className="space-y-2">
        <Label htmlFor="name">名称</Label>
        <Input
          id="name"
          value={formData.name}
          onChange={(e) => onChange({ ...formData, name: e.target.value })}
          placeholder="例如: OpenAI"
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="base_url">接口地址</Label>
        <Input
          id="base_url"
          value={formData.base_url}
          onChange={(e) => onChange({ ...formData, base_url: e.target.value })}
          placeholder="例如: https://api.openai.com"
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="api_key">API Key</Label>
        <Input
          id="api_key"
          type="password"
          value={formData.api_key || ""}
          onChange={(e) => onChange({ ...formData, api_key: e.target.value })}
          placeholder="sk-..."
        />
      </div>
      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-2">
          <Label htmlFor="priority">优先级</Label>
          <Input
            id="priority"
            type="number"
            value={formData.priority}
            onChange={(e) =>
              onChange({ ...formData, priority: parseInt(e.target.value) || 0 })
            }
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="weight">权重</Label>
          <Input
            id="weight"
            type="number"
            value={formData.weight}
            onChange={(e) =>
              onChange({ ...formData, weight: parseInt(e.target.value) || 1 })
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
