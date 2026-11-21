"use client"

import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import { AccountCreateUpdate } from "@/api/account"
import {
  LCheckbox,
  LSelect,
  LSelectOption,
} from "@/components/widgets/form-fields"
import { roleAllListApi } from "@/api/roles"
import { useEffect, useState } from "react"

interface AccountFormProps {
  formData: AccountCreateUpdate
  onChange: (data: AccountCreateUpdate) => void
}

export function AccountForm({ formData, onChange }: AccountFormProps) {
  const [roleOptions, setRoleOptions] = useState<LSelectOption[]>([])
  useEffect(() => {
    let cancelled = false
    roleAllListApi()
      .then((roles) => {
        if (!cancelled) {
          setRoleOptions(
            roles.map((role) => ({
              value: role.id,
              label: role.name,
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
        <Label htmlFor="name">姓名</Label>
        <Input
          id="name"
          value={formData.name}
          onChange={(e) => onChange({ ...formData, name: e.target.value })}
          placeholder="请输入姓名"
          suppressHydrationWarning
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="email">邮箱</Label>
        <Input
          id="email"
          type="email"
          value={formData.email}
          onChange={(e) => onChange({ ...formData, email: e.target.value })}
          placeholder="请输入邮箱"
          suppressHydrationWarning
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="password">密码</Label>
        <Input
          id="password"
          type="password"
          value={formData.password || ""}
          onChange={(e) => onChange({ ...formData, password: e.target.value })}
          placeholder="请输入密码"
          suppressHydrationWarning
        />
      </div>
      <div className="space-y-2">
        <LSelect
          id="roles"
          label="角色"
          value={formData.roles.map((role) => role.id)}
          placeholder="请选择角色"
          options={roleOptions}
          onChange={(value) => {
            const ids = Array.isArray(value) ? value : [value]
            const roles = ids.map((id) => {
              const option = roleOptions.find((opt) => opt.value === id)
              return {
                id: id as number,
                name: option?.label || "",
              }
            })
            onChange({ ...formData, roles: roles })
          }}
          multiSelect={true}
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
