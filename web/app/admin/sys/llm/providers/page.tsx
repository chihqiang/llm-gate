"use client"

import { useState } from "react"
import {
  Provider,
  UpstreamModel,
  providerListApi,
  providerCreateApi,
  providerUpdateApi,
  providerDeleteApi,
  providerDetailApi,
  providerPreviewSyncModelsApi,
  providerSyncModelsApi,
} from "@/api/llm"
import { ProviderForm } from "@/components/forms/provider-form"
import { Crud, SearchField } from "@/components/widgets/crud"
import { DataListColumn } from "@/components/widgets/data-list"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "@/components/ui/dialog"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { toast } from "sonner"
import { RefreshCw } from "lucide-react"

const searchFields: SearchField[] = [
  {
    name: "name",
    label: "服务商",
    type: "input",
    placeholder: "搜索服务商名称",
  },
]

const columns: DataListColumn<Provider>[] = [
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
    key: "base_url",
    header: "接口地址",
    cell: (row) => row.base_url,
  },
  {
    key: "priority",
    header: "优先级",
    cell: (row) => row.priority,
  },
  {
    key: "weight",
    header: "权重",
    cell: (row) => row.weight,
  },
  {
    key: "status",
    header: "状态",
    cell: (row) => (row.status ? "启用" : "禁用"),
  },
]

const defaultFormData: Provider = {
  id: 0,
  name: "",
  base_url: "",
  api_key: "",
  status: true,
  priority: 0,
  weight: 1,
  remark: "",
  created_at: "",
  updated_at: "",
}

export default function ProvidersPage() {
  const [loading, setLoading] = useState(false)
  const [syncing, setSyncing] = useState(false)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [currentProvider, setCurrentProvider] = useState<Provider | null>(null)
  const [upstreamModels, setUpstreamModels] = useState<UpstreamModel[]>([])
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [search, setSearch] = useState("")

  const openSyncDialog = async (provider: Provider) => {
    setCurrentProvider(provider)
    setDialogOpen(true)
    setLoading(true)
    setSearch("")
    try {
      const preview = await providerPreviewSyncModelsApi(provider.id)
      setUpstreamModels(preview.models)
      setSelected(new Set(preview.models.filter((m) => !m.exists).map((m) => m.id)))
    } catch {
      toast.error("获取上游模型列表失败")
      setDialogOpen(false)
    } finally {
      setLoading(false)
    }
  }

  const handleSync = async () => {
    if (!currentProvider) return
    setSyncing(true)
    try {
      const result = await providerSyncModelsApi(currentProvider.id, Array.from(selected))
      toast.success(`${currentProvider.name}: 同步完成，新增 ${result.created} 个，跳过 ${result.skipped} 个`)
      setDialogOpen(false)
    } catch {
      toast.error(`${currentProvider.name}: 同步失败`)
    } finally {
      setSyncing(false)
    }
  }

  const toggleAll = (checked: boolean) => {
    const filtered = filteredModels.map((m) => m.id)
    if (checked) {
      setSelected(new Set([...selected, ...filtered]))
    } else {
      const next = new Set(selected)
      filtered.forEach((id) => next.delete(id))
      setSelected(next)
    }
  }

  const toggleOne = (id: string, checked: boolean) => {
    const next = new Set(selected)
    if (checked) {
      next.add(id)
    } else {
      next.delete(id)
    }
    setSelected(next)
  }

  const filteredModels = upstreamModels.filter(
    (m) => !search || m.id.toLowerCase().includes(search.toLowerCase())
  )
  const allFilteredSelected = filteredModels.every((m) => selected.has(m.id))

  return (
    <>
      <Crud<Provider, Provider>
        title="服务商管理"
        entityName="服务商"
        columns={columns}
        listApi={providerListApi}
        pageSize={8}
        searchFields={searchFields}
        deleteApi={providerDeleteApi}
        formComponent={ProviderForm}
        defaultFormData={defaultFormData}
        createApi={providerCreateApi}
        updateApi={providerUpdateApi}
        detailApi={providerDetailApi}
        extraActions={(item) => (
          <Button
            variant="ghost"
            size="icon-sm"
            className="text-muted-foreground hover:text-foreground"
            onClick={(e) => {
              e.stopPropagation()
              openSyncDialog(item)
            }}
            title="同步模型"
          >
            <RefreshCw className="size-3.5" />
          </Button>
        )}
      />

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="max-w-lg max-h-[80vh] flex flex-col">
          <DialogHeader>
            <DialogTitle>同步模型 - {currentProvider?.name}</DialogTitle>
            <DialogDescription>选择要同步到本地的上游模型</DialogDescription>
          </DialogHeader>

          <div className="flex items-center gap-2 py-2">
            <Input
              placeholder="搜索模型..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>

          <div className="flex-1 overflow-y-auto min-h-0 space-y-1">
            {loading ? (
              <div className="text-center text-muted-foreground py-8">加载中...</div>
            ) : filteredModels.length === 0 ? (
              <div className="text-center text-muted-foreground py-8">
                {search ? "无匹配模型" : "暂无上游模型"}
              </div>
            ) : (
              <>
                <div className="flex items-center gap-2 px-1 pb-1 border-b">
                  <Checkbox
                    id="select-all"
                    checked={filteredModels.length > 0 && allFilteredSelected}
                    onCheckedChange={(checked) => toggleAll(checked === true)}
                  />
                  <Label htmlFor="select-all" className="text-sm text-muted-foreground">
                    全选当前列表
                  </Label>
                </div>
                {filteredModels.map((m) => (
                  <div
                    key={m.id}
                    className="flex items-center gap-2 px-1 py-1.5 hover:bg-muted rounded-sm"
                  >
                    <Checkbox
                      id={`model-${m.id}`}
                      checked={selected.has(m.id)}
                      onCheckedChange={(checked) => toggleOne(m.id, checked === true)}
                    />
                    <Label
                      htmlFor={`model-${m.id}`}
                      className={`text-sm flex-1 cursor-pointer ${m.exists ? "text-muted-foreground line-through" : ""}`}
                    >
                      {m.id}
                      {m.exists && (
                        <span className="ml-2 text-xs text-muted-foreground">(已存在)</span>
                      )}
                    </Label>
                  </div>
                ))}
              </>
            )}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setDialogOpen(false)}>
              取消
            </Button>
            <Button onClick={handleSync} disabled={syncing || selected.size === 0}>
              {syncing ? "同步中..." : `同步 (${selected.size})`}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
