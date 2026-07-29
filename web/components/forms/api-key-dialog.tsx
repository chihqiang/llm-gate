"use client"

import { useState, useEffect, useCallback } from "react"
import {
  UserToken,
  tokenListApi,
  tokenCreateApi,
  tokenDeleteApi,
} from "@/api/llm"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { toast } from "sonner"
import { Copy, Plus, Trash2, KeyIcon } from "lucide-react"

interface ApiKeyDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  accountId: number
}

export function ApiKeyDialog({
  open,
  onOpenChange,
  accountId,
}: ApiKeyDialogProps) {
  const [tokens, setTokens] = useState<UserToken[]>([])
  const [loading, setLoading] = useState(false)
  const [newName, setNewName] = useState("")
  const [creating, setCreating] = useState(false)

  const fetchTokens = useCallback(async () => {
    setLoading(true)
    try {
      const res = await tokenListApi({
        page: 1,
        size: 50,
        account_id: accountId,
      })
      setTokens(res.data)
    } catch {
      toast.error("获取 Key 列表失败")
    } finally {
      setLoading(false)
    }
  }, [accountId])

  useEffect(() => {
    if (open) {
      fetchTokens()
      setNewName("")
    }
  }, [open, fetchTokens])

  const handleCreate = async () => {
    if (!newName.trim()) {
      toast.error("请输入名称")
      return
    }
    setCreating(true)
    try {
      await tokenCreateApi({
        account_id: accountId,
        name: newName.trim(),
        quota: 100000,
        status: true,
      })
      toast.success("创建成功")
      setNewName("")
      await fetchTokens()
    } catch {
      toast.error("创建失败")
    } finally {
      setCreating(false)
    }
  }

  const handleDelete = async (id: number) => {
    try {
      await tokenDeleteApi(id)
      toast.success("已删除")
      await fetchTokens()
    } catch {
      toast.error("删除失败")
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="flex max-h-[80vh] max-w-lg flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <KeyIcon className="size-5" />
            API Key 管理
          </DialogTitle>
        </DialogHeader>

        <div className="flex items-end gap-2">
          <div className="flex-1 space-y-1">
            <Label htmlFor="new-key-name">新建 Key</Label>
            <Input
              id="new-key-name"
              placeholder="输入名称后回车"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") handleCreate()
              }}
            />
          </div>
          <Button onClick={handleCreate} disabled={creating || !newName.trim()}>
            <Plus className="mr-1 size-4" />
            创建
          </Button>
        </div>

        <div className="mt-4 min-h-0 flex-1 space-y-2 overflow-y-auto">
          {loading ? (
            <div className="py-8 text-center text-muted-foreground">
              加载中...
            </div>
          ) : tokens.length === 0 ? (
            <div className="py-8 text-center text-muted-foreground">
              暂无 Key
            </div>
          ) : (
            tokens.map((t) => (
              <div
                key={t.id}
                className="flex items-center gap-2 rounded-lg border p-3"
              >
                <div className="min-w-0 flex-1">
                  <div className="text-sm font-medium">{t.name}</div>
                  <div className="mt-1 flex items-center gap-1">
                    <code className="truncate rounded bg-muted px-1.5 py-0.5 font-mono text-xs">
                      {t.key}
                    </code>
                    <button
                      type="button"
                      onClick={() => {
                        navigator.clipboard.writeText(t.key)
                        toast.success("已复制")
                      }}
                      className="shrink-0 text-muted-foreground hover:text-foreground"
                    >
                      <Copy className="size-3.5" />
                    </button>
                  </div>
                  <div className="mt-1 text-xs text-muted-foreground">
                    配额: {t.quota.toLocaleString()}
                    {t.status ? "" : " (已禁用)"}
                  </div>
                </div>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  className="shrink-0 text-muted-foreground hover:text-destructive"
                  onClick={() => handleDelete(t.id)}
                >
                  <Trash2 className="size-4" />
                </Button>
              </div>
            ))
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
