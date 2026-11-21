"use client"

import React from "react"
import { X, RefreshCw, XCircle, Square } from "lucide-react"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuTrigger,
} from "@/components/ui/context-menu"
import type { TabItem } from "@/hooks/use-tabs"

// -----------------------------------------------------------------------------
// 类型定义
// -----------------------------------------------------------------------------

interface TabBarProps {
  tabs: TabItem[]
  activePath: string
  onNavigate: (path: string) => void
  onClose: (path: string) => void
  onCloseOthers: (path: string) => void
  onCloseAll: () => void
  onRefresh: (path: string) => void
}

// -----------------------------------------------------------------------------
// 单个标签项
// -----------------------------------------------------------------------------

interface TabItemProps {
  tab: TabItem
  isActive: boolean
  onNavigate: () => void
  onClose: () => void
  onRefresh: () => void
  onCloseOthers: () => void
  onCloseAll: () => void
}

function TabBarItem({
  tab,
  isActive,
  onNavigate,
  onClose,
  onRefresh,
  onCloseOthers,
  onCloseAll,
}: TabItemProps) {
  // 中键点击关闭
  const handleAuxClick = (e: React.MouseEvent) => {
    if (e.button === 1 && tab.closable) {
      e.preventDefault()
      onClose()
    }
  }

  return (
    <ContextMenu>
      <ContextMenuTrigger
        render={
          <button
            type="button"
            data-active={isActive || undefined}
            onClick={onNavigate}
            onAuxClick={handleAuxClick}
            className={cn(
              "group relative flex h-8 shrink-0 items-center gap-1.5 rounded-md px-3 text-sm transition-colors",
              isActive
                ? "bg-primary/10 text-primary"
                : "text-muted-foreground hover:bg-muted hover:text-foreground"
            )}
          >
            {/* 激活指示条 */}
            {isActive && (
              <span className="absolute top-0 left-1/2 h-0.5 w-6 -translate-x-1/2 rounded-full bg-primary" />
            )}
            <span className="max-w-[120px] truncate">{tab.title}</span>
            {tab.closable && (
              <span
                role="button"
                tabIndex={-1}
                onClick={(e) => {
                  e.stopPropagation()
                  onClose()
                }}
                onKeyDown={(e) => {
                  if (e.key === "Enter" || e.key === " ") {
                    e.stopPropagation()
                    onClose()
                  }
                }}
                className="ml-0.5 flex h-4 w-4 shrink-0 items-center justify-center rounded-sm opacity-0 transition-opacity group-hover:opacity-100 hover:bg-muted-foreground/20"
                aria-label="关闭标签"
              >
                <X className="h-3 w-3" />
              </span>
            )}
            {/* 不可关闭的首页 Tab 激活时显示圆点 */}
            {!tab.closable && isActive && (
              <span className="ml-1 h-1.5 w-1.5 shrink-0 rounded-full bg-primary" />
            )}
          </button>
        }
      />
      <ContextMenuContent>
        <ContextMenuItem onClick={onRefresh} disabled={!isActive}>
          <RefreshCw className="mr-2 h-4 w-4" />
          刷新
        </ContextMenuItem>
        <ContextMenuItem onClick={onClose} disabled={!tab.closable}>
          <X className="mr-2 h-4 w-4" />
          关闭
        </ContextMenuItem>
        <ContextMenuItem onClick={onCloseOthers}>
          <XCircle className="mr-2 h-4 w-4" />
          关闭其他
        </ContextMenuItem>
        <ContextMenuSeparator />
        <ContextMenuItem onClick={onCloseAll}>
          <Square className="mr-2 h-4 w-4" />
          关闭所有
        </ContextMenuItem>
      </ContextMenuContent>
    </ContextMenu>
  )
}

// -----------------------------------------------------------------------------
// TabBar 主组件
// -----------------------------------------------------------------------------

export function TabBar({
  tabs,
  activePath,
  onNavigate,
  onClose,
  onCloseOthers,
  onCloseAll,
  onRefresh,
}: TabBarProps) {
  const scrollRef = React.useRef<HTMLDivElement>(null)

  // 滚动到激活的 Tab
  React.useEffect(() => {
    const container = scrollRef.current
    if (!container) return

    const activeEl =
      container.querySelector<HTMLButtonElement>(`[data-active="true"]`)
    if (activeEl) {
      activeEl.scrollIntoView({
        behavior: "smooth",
        block: "nearest",
        inline: "nearest",
      })
    }
  }, [activePath])

  return (
    <div className="flex h-9 items-center border-b bg-background">
      {/* 可滚动的 Tab 区域 */}
      <div
        ref={scrollRef}
        className="flex flex-1 items-center gap-0.5 overflow-x-auto scroll-smooth px-2 [&::-webkit-scrollbar]:hidden"
        style={{ scrollbarWidth: "none" }}
      >
        {tabs.map((tab) => (
          <TabBarItem
            key={tab.path}
            tab={tab}
            isActive={tab.path === activePath}
            onNavigate={() => onNavigate(tab.path)}
            onClose={() => onClose(tab.path)}
            onRefresh={() => onRefresh(tab.path)}
            onCloseOthers={() => onCloseOthers(tab.path)}
            onCloseAll={onCloseAll}
          />
        ))}
      </div>

      {/* 右侧操作：关闭全部 */}
      {tabs.length > 1 && (
        <Button
          variant="ghost"
          size="sm"
          className="mr-1 h-7 shrink-0 gap-1 px-2 text-xs text-muted-foreground"
          onClick={onCloseAll}
          title="关闭所有标签页"
        >
          <Square className="h-3.5 w-3.5" />
          关闭全部
        </Button>
      )}
    </div>
  )
}
