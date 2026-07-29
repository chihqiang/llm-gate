"use client"

import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Checkbox } from "@/components/ui/checkbox"
import { RoleFromRequest } from "@/api/roles"

interface RoleFormProps {
  formData: RoleFromRequest
  onChange: (data: RoleFromRequest) => void
}

export function RoleForm({ formData, onChange }: RoleFormProps) {
  return (
    <div className="space-y-4 py-4">
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="name">角色名称</Label>
          <Input
            id="name"
            value={formData.name}
            onChange={(e) => onChange({ ...formData, name: e.target.value })}
            placeholder="请输入角色名称"
            suppressHydrationWarning
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="sort">排序</Label>
          <Input
            id="sort"
            type="number"
            value={formData.sort}
            onChange={(e) =>
              onChange({ ...formData, sort: parseInt(e.target.value) || 0 })
            }
            placeholder="请输入排序值"
            suppressHydrationWarning
          />
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="remark">备注</Label>
        <Textarea
          id="remark"
          value={formData.remark}
          onChange={(e) => onChange({ ...formData, remark: e.target.value })}
          placeholder="请输入角色备注"
          rows={3}
          suppressHydrationWarning
        />
      </div>

      <div className="space-y-2">
        <div className="flex items-center space-x-2">
          <Checkbox
            id="status"
            checked={formData.status}
            onCheckedChange={(checked) =>
              onChange({ ...formData, status: checked || false })
            }
          />
          <Label htmlFor="status">启用角色</Label>
        </div>
      </div>
    </div>
  )
}
