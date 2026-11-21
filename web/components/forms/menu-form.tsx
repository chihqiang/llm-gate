"use client"

import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { IconSelector } from "@/components/widgets/icon"
import { Menu, menuAllApi, menuTypeMap, apiMethodMap } from "@/api/menu"
import { useEffect, useState } from "react"

interface MenuFormProps {
  formData: Menu
  onChange: (data: Menu) => void
}

export function MenuForm({ formData, onChange }: MenuFormProps) {
  const [menus, setMenus] = useState<Menu[]>([])

  useEffect(() => {
    let cancelled = false
    menuAllApi()
      .then((data) => {
        if (!cancelled) setMenus(data)
      })
      .catch(() => {})
    return () => {
      cancelled = true
    }
  }, [])

  const parentMenus = menus.filter((menu) => menu.menu_type === 1)
  const menuTypeVal = formData.menu_type ?? 1
  const pidVal = formData.pid?.toString() || "0"
  const apiMethodVal = formData.api_method ?? ""

  const getParentLabel = () => {
    if (pidVal === "0") return "根菜单"
    const found = parentMenus.find((m) => m.id.toString() === pidVal)
    return found?.name || "根菜单"
  }

  return (
    <div className="space-y-4 py-4">
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="name">菜单名称</Label>
          <Input
            id="name"
            value={formData.name}
            onChange={(e) => onChange({ ...formData, name: e.target.value })}
            placeholder="请输入菜单名称"
            suppressHydrationWarning
          />
        </div>

        <div className="space-y-2">
          <Label>菜单类型</Label>
          <Select
            value={String(menuTypeVal)}
            onValueChange={(val) => {
              onChange({ ...formData, menu_type: Number(val) })
            }}
          >
            <SelectTrigger>
              <SelectValue>{menuTypeMap[menuTypeVal]}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              {Object.entries(menuTypeMap).map(([value, label]) => (
                <SelectItem key={value} value={value}>
                  {label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div className="space-y-2">
          <Label>父菜单</Label>
          <Select
            value={pidVal}
            onValueChange={(val) => {
              onChange({ ...formData, pid: Number(val) })
            }}
          >
            <SelectTrigger>
              <SelectValue>{getParentLabel()}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="0">根菜单</SelectItem>
              {parentMenus.map((menu) => (
                <SelectItem key={menu.id} value={String(menu.id)}>
                  {menu.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="space-y-2">
          <Label>排序</Label>
          <Input
            type="number"
            value={formData.sort}
            onChange={(e) =>
              onChange({ ...formData, sort: parseInt(e.target.value) || 0 })
            }
            suppressHydrationWarning
          />
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div className="space-y-2">
          <Label>路由地址</Label>
          <Input
            value={formData.path}
            onChange={(e) => onChange({ ...formData, path: e.target.value })}
            placeholder="请输入路由地址"
            suppressHydrationWarning
          />
        </div>
        <div className="space-y-2">
          <Label>组件路径</Label>
          <Input
            value={formData.component}
            onChange={(e) =>
              onChange({ ...formData, component: e.target.value })
            }
            placeholder="请输入组件路径"
            suppressHydrationWarning
          />
        </div>
      </div>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div className="space-y-2">
          <Label>菜单图标</Label>
          <IconSelector
            value={formData.icon || ""}
            onChange={(value) => onChange({ ...formData, icon: value })}
          />
        </div>

        <div className="space-y-2">
          <Label>请求方式</Label>
          <Select
            value={apiMethodVal}
            onValueChange={(val) =>
              onChange({ ...formData, api_method: val || "" })
            }
          >
            <SelectTrigger>
              <SelectValue>{apiMethodMap[apiMethodVal]}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              {Object.entries(apiMethodMap).map(([value, label]) => (
                <SelectItem key={value} value={value}>
                  {label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="space-y-2">
        <Label>接口地址</Label>
        <Input
          value={formData.api_url}
          onChange={(e) => onChange({ ...formData, api_url: e.target.value })}
          placeholder="请输入接口地址"
          suppressHydrationWarning
        />
      </div>

      <div className="space-y-2">
        <Label>备注</Label>
        <Textarea
          value={formData.remark}
          onChange={(e) => onChange({ ...formData, remark: e.target.value })}
          rows={3}
          suppressHydrationWarning
        />
      </div>

      <div className="flex items-center gap-6">
        <div className="flex items-center space-x-2">
          <Checkbox
            checked={formData.visible}
            onCheckedChange={(v) => onChange({ ...formData, visible: !!v })}
          />
          <Label>是否显示</Label>
        </div>
        <div className="flex items-center space-x-2">
          <Checkbox
            checked={formData.status}
            onCheckedChange={(v) => onChange({ ...formData, status: !!v })}
          />
          <Label>启用状态</Label>
        </div>
      </div>
    </div>
  )
}
