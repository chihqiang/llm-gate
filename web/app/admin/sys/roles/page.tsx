"use client"

import { useState, useCallback } from "react"
import { Shield } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import {
  Role,
  RoleFromRequest,
  roleListApi,
  roleCreateApi,
  roleUpdateApi,
  roleDeleteApi,
  roleDetailApi,
  roleAssociateMenusApi,
} from "@/api/roles"
import { RoleForm } from "@/components/forms/role-form"
import { RoleAuthorizeDialog } from "@/components/forms/role-authorize-dialog"
import { Crud, SearchField } from "@/components/widgets/crud"
import { DataListColumn } from "@/components/widgets/data-list"

const searchFields: SearchField[] = [
  {
    name: "id",
    label: "角色ID",
    type: "input",
    placeholder: "搜索角色ID",
  },
]

const defaultFormData: RoleFromRequest = {
  id: 0,
  name: "",
  sort: 0,
  status: true,
  remark: "",
  menus: [],
}

export default function RolesPage() {
  // 授权弹窗状态
  const [isAuthorizeOpen, setIsAuthorizeOpen] = useState(false)
  const [authorizeRole, setAuthorizeRole] = useState<Role | null>(null)
  const [refreshKey, setRefreshKey] = useState(0)
  const [isAuthorizing, setIsAuthorizing] = useState(false)

  // 关闭授权弹窗
  const closeAuthorize = useCallback(() => {
    setIsAuthorizeOpen(false)
    setAuthorizeRole(null)
  }, [])

  // 处理授权
  const handleAuthorize = useCallback(
    async (roleId: number, menuIds: number[]) => {
      setIsAuthorizing(true)
      try {
        await roleAssociateMenusApi(roleId, menuIds)
        closeAuthorize()
        setRefreshKey((k) => k + 1)
        toast.success("授权成功")
      } catch (error) {
        console.error("授权失败", error)
        toast.error("授权失败，请稍后重试")
      } finally {
        setIsAuthorizing(false)
      }
    },
    [closeAuthorize]
  )

  const columns: DataListColumn<Role>[] = [
    {
      key: "id",
      header: "ID",
      cellClassName: "w-16 font-medium",
      cell: (row) => row.id,
    },
    {
      key: "name",
      header: "角色名称",
      cell: (row) => (
        <div className="flex items-center gap-2">
          <Shield className="h-4 w-4 text-blue-500" />
          <span className="font-medium">{row.name}</span>
        </div>
      ),
    },
    {
      key: "sort",
      header: "排序",
      cell: (row) => row.sort,
    },
    {
      key: "remark",
      header: "备注",
      cell: (row) => row.remark || "-",
    },
    {
      key: "status",
      header: "状态",
      cell: (row) => (row.status ? "正常" : "禁用"),
    },
  ]

  return (
    <>
      <Crud<Role, RoleFromRequest>
        key={refreshKey}
        title="角色管理"
        entityName="角色"
        columns={columns}
        listApi={roleListApi}
        pageSize={8}
        searchFields={searchFields}
        deleteApi={roleDeleteApi}
        createPermission="/api/v1/sys/roles"
        updatePermission="/api/v1/sys/roles/*"
        deletePermission="/api/v1/sys/roles/*"
        formComponent={RoleForm}
        defaultFormData={defaultFormData}
        createApi={roleCreateApi}
        updateApi={roleUpdateApi}
        detailApi={roleDetailApi}
        extraActions={(row) => (
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={(e) => {
              e.stopPropagation()
              setAuthorizeRole(row)
              setIsAuthorizeOpen(true)
            }}
            title="授权"
          >
            <Shield className="h-4 w-4" />
          </Button>
        )}
      />

      {/* 授权弹窗 */}
      <RoleAuthorizeDialog
        open={isAuthorizeOpen}
        onOpenChange={closeAuthorize}
        role={authorizeRole}
        onAuthorize={handleAuthorize}
        isLoading={isAuthorizing}
      />
    </>
  )
}
