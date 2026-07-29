"use client"

import { useCallback, useEffect, useState } from "react"
import { usePathname, useRouter } from "next/navigation"
import { getPageTitle } from "@/lib/nav"
import { local } from "@/lib/storage"

// -----------------------------------------------------------------------------
// 类型定义
// -----------------------------------------------------------------------------

export interface TabItem {
  /** 路由路径，唯一标识 */
  path: string
  /** 标签页显示标题 */
  title: string
  /** 是否可关闭（首页 Tab 不可关闭） */
  closable: boolean
}

interface TabState {
  tabs: TabItem[]
  activePath: string
}

// -----------------------------------------------------------------------------
// 常量
// -----------------------------------------------------------------------------

const STORAGE_KEY = "ADMIN_TABS"

/** 默认首页 Tab（不可关闭） */
const HOME_TAB: TabItem = {
  path: "/admin/dashboard",
  title: "控制台",
  closable: false,
}

// -----------------------------------------------------------------------------
// localStorage 读取（仅在客户端执行）
// -----------------------------------------------------------------------------

function readSavedTabs(): TabItem[] {
  if (typeof window === "undefined") return [HOME_TAB]
  const saved = local.get<TabState | null>(STORAGE_KEY, null)
  if (saved?.tabs?.length) {
    const hasHome = saved.tabs.some((t) => t.path === HOME_TAB.path)
    return hasHome ? saved.tabs : [HOME_TAB, ...saved.tabs]
  }
  return [HOME_TAB]
}

// -----------------------------------------------------------------------------
// Hook 实现
// -----------------------------------------------------------------------------

/**
 * 多标签页管理 Hook
 *
 * 功能：
 * - 自动监听路由变化，打开/激活对应 Tab
 * - Tab 持久化到 localStorage，刷新后恢复
 * - 支持关闭单个、关闭其他、关闭全部
 * - 首页 Tab 不可关闭
 */
export function useTabs() {
  const router = useRouter()
  const pathname = usePathname()

  // 懒初始化：从 localStorage 恢复 Tab 状态（SSR 时返回默认值）
  const [tabs, setTabs] = useState<TabItem[]>(() => readSavedTabs())
  const [activePath, setActivePath] = useState(pathname)

  // 路由变化时自动添加/激活 Tab（在渲染期间调整状态，避免 effect 中 setState）
  const [prevPathname, setPrevPathname] = useState(pathname)
  if (pathname !== prevPathname) {
    setPrevPathname(pathname)
    if (pathname.startsWith("/admin")) {
      const title = getPageTitle(pathname)
      const isHome = pathname === HOME_TAB.path
      setTabs((prev) => {
        const existing = prev.find((t) => t.path === pathname)
        if (existing) {
          // 已存在，更新标题（可能动态菜单变了）
          return prev.map((t) => (t.path === pathname ? { ...t, title } : t))
        }
        // 新增 Tab
        return [...prev, { path: pathname, title, closable: !isHome }]
      })
    }
    setActivePath(pathname)
  }

  // 持久化到 localStorage（更新外部系统，符合 effect 用途）
  useEffect(() => {
    local.set<TabState>(STORAGE_KEY, { tabs, activePath })
  }, [tabs, activePath])

  // 点击 Tab 导航
  const navigateToTab = useCallback(
    (path: string) => {
      if (path === pathname) return
      router.push(path)
    },
    [router, pathname]
  )

  // 关闭单个 Tab
  const closeTab = useCallback(
    (path: string) => {
      let navigateTo: string | null = null
      setTabs((prev) => {
        const target = prev.find((t) => t.path === path)
        if (!target || !target.closable) return prev

        const newTabs = prev.filter((t) => t.path !== path)

        if (path === activePath) {
          const closedIndex = prev.findIndex((t) => t.path === path)
          const nextActive = newTabs[Math.min(closedIndex, newTabs.length - 1)]
          if (nextActive) {
            navigateTo = nextActive.path
          }
        }

        return newTabs
      })
      if (navigateTo) {
        router.push(navigateTo)
      }
    },
    [activePath, router]
  )

  // 关闭其他 Tab
  const closeOtherTabs = useCallback(
    (path: string) => {
      let needNavigate = path !== pathname
      setTabs((prev) => {
        const target = prev.find((t) => t.path === path)
        if (!target) return prev
        return prev.filter((t) => t.path === path || !t.closable)
      })
      if (needNavigate) {
        router.push(path)
      }
    },
    [router, pathname]
  )

  // 关闭所有可关闭的 Tab（保留首页）
  const closeAllTabs = useCallback(() => {
    setTabs((prev) => prev.filter((t) => !t.closable))
    // 跳回首页
    if (pathname !== HOME_TAB.path) {
      router.push(HOME_TAB.path)
    }
  }, [router, pathname])

  // 刷新当前 Tab
  const refreshTab = useCallback(
    (path: string) => {
      if (path === pathname) {
        router.refresh()
      } else {
        router.push(path)
      }
    },
    [router, pathname]
  )

  return {
    tabs,
    activePath,
    navigateToTab,
    closeTab,
    closeOtherTabs,
    closeAllTabs,
    refreshTab,
  }
}
