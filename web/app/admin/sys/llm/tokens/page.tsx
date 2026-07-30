"use client"

import { useState } from "react"
import {
  UserToken,
  tokenListApi,
  tokenCreateApi,
  tokenUpdateApi,
  tokenDeleteApi,
  tokenDetailApi,
  tokenRevealApi,
} from "@/api/llm"
import { TokenForm } from "@/components/forms/token-form"
import { Crud, SearchField } from "@/components/widgets/crud"
import { DataListColumn } from "@/components/widgets/data-list"
import { Copy, Check } from "lucide-react"
import { toast } from "sonner"

const searchFields: SearchField[] = [
  {
    name: "name",
    label: "名称",
    type: "input",
    placeholder: "搜索名称",
  },
]

function CopyButton({ tokenId }: { tokenId: number }) {
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    try {
      const key = await tokenRevealApi(tokenId)
      await navigator.clipboard.writeText(key)
      setCopied(true)
      toast.success("已复制到剪贴板")
      setTimeout(() => setCopied(false), 2000)
    } catch {
      toast.error("复制失败")
    }
  }

  return (
    <button
      type="button"
      onClick={handleCopy}
      className="shrink-0 text-muted-foreground hover:text-foreground"
      title="复制 Key"
    >
      {copied ? (
        <Check className="h-3.5 w-3.5" />
      ) : (
        <Copy className="h-3.5 w-3.5" />
      )}
    </button>
  )
}

const columns: DataListColumn<UserToken>[] = [
  {
    key: "id",
    header: "ID",
    cellClassName: "w-16 font-medium",
    cell: (row) => row.id,
  },
  {
    key: "name",
    header: "名称",
    cell: (row) => row.name,
  },
  {
    key: "key_masked",
    header: "Key",
    cell: (row) => (
      <div className="flex items-center gap-1.5">
        <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs">
          {row.key_masked}
        </code>
        <CopyButton tokenId={row.id} />
      </div>
    ),
  },
  {
    key: "account_id",
    header: "账号ID",
    cell: (row) => row.account_id,
  },
  {
    key: "quota",
    header: "剩余配额",
    cell: (row) => row.quota.toLocaleString(),
  },
  {
    key: "status",
    header: "状态",
    cell: (row) => (row.status ? "启用" : "禁用"),
  },
]

const defaultFormData: UserToken = {
  id: 0,
  account_id: 0,
  name: "",
  key: "",
  key_masked: "",
  quota: 0,
  status: true,
  expired_at: null,
  created_at: "",
  updated_at: "",
}

export default function TokensPage() {
  return (
    <Crud<UserToken, UserToken>
      title="API Key 管理"
      entityName="API Key"
      columns={columns}
      listApi={tokenListApi}
      pageSize={8}
      searchFields={searchFields}
      deleteApi={tokenDeleteApi}
      formComponent={TokenForm}
      defaultFormData={defaultFormData}
      createApi={tokenCreateApi}
      updateApi={tokenUpdateApi}
      detailApi={tokenDetailApi}
    />
  )
}
