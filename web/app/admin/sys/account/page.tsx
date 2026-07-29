"use client"

import { Account, AccountCreateUpdate } from "@/api/account"
import {
  accountListApi,
  accountCreateApi,
  accountUpdateApi,
  accountDeleteApi,
  accountDetailApi,
} from "@/api/account"
import { AccountForm } from "@/components/forms/account-form"
import { Crud, SearchField } from "@/components/widgets/crud"
import { DataListColumn } from "@/components/widgets/data-list"

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
}

export default function AccountPage() {
  return (
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
    />
  )
}
