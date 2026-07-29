"use client"

import {
  Menu,
  menuListApi,
  menuCreateApi,
  menuUpdateApi,
  menuDeleteApi,
  menuDetailApi,
  menuTypeMap,
} from "@/api/menu"
import { MenuForm } from "@/components/forms/menu-form"
import { Crud, SearchField } from "@/components/widgets/crud"
import { DataListColumn } from "@/components/widgets/data-list"

const searchFields: SearchField[] = [
  {
    name: "id",
    label: "菜单ID",
    type: "input",
    placeholder: "搜索菜单ID",
  },
]

const defaultFormData: Menu = {
  id: 0,
  pid: 0,
  menu_type: 1,
  name: "",
  path: "",
  component: "",
  icon: "",
  sort: 0,
  api_url: "",
  api_method: "",
  visible: true,
  status: true,
  remark: "",
}

export default function MenuPage() {
  const columns: DataListColumn<Menu>[] = [
    {
      key: "id",
      header: "ID",
      cellClassName: "w-16 font-medium",
      cell: (row) => row.id,
    },
    {
      key: "name",
      header: "菜单名称",
      cell: (row) => row.name,
    },
    {
      key: "menu_type",
      header: "类型",
      cell: (row) => menuTypeMap[row.menu_type] || "未知",
    },
    {
      key: "path",
      header: "路由",
      cell: (row) => row.path || "-",
    },
    {
      key: "api_url",
      header: "接口地址",
      cell: (row) => row.api_url || "-",
    },
    {
      key: "visible",
      header: "显示",
      cell: (row) => (row.visible ? "显示" : "隐藏"),
    },
    {
      key: "status",
      header: "状态",
      cell: (row) => (row.status ? "正常" : "禁用"),
    },
  ]

  return (
    <Crud<Menu, Menu>
      title="菜单管理"
      entityName="菜单"
      columns={columns}
      listApi={menuListApi}
      pageSize={8}
      searchFields={searchFields}
      deleteApi={menuDeleteApi}
      createPermission="/api/v1/sys/menus"
      updatePermission="/api/v1/sys/menus/*"
      deletePermission="/api/v1/sys/menus/*"
      formComponent={MenuForm}
      defaultFormData={defaultFormData}
      createApi={menuCreateApi}
      updateApi={menuUpdateApi}
      detailApi={menuDetailApi}
    />
  )
}
