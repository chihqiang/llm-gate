"use client"

import React, { useEffect, useMemo } from "react"
import Link from "next/link"
import { usePathname } from "next/navigation"
import {
  Building2,
  ChevronDown,
  ChevronRight,
  Moon,
  Sun,
  Search,
  X,
} from "lucide-react"
import { useTheme } from "next-themes"

import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubItem,
  SidebarProvider,
  SidebarRail,
  SidebarTrigger,
  SidebarFooter,
} from "@/components/ui/sidebar"
import { useState } from "react"
import { AuthMenuTree } from "@/lib/account"
import { NavAccount } from "@/components/layout/nav-account"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { useAuth } from "@/hooks/use-auth"
import { Icon } from "@/components/widgets/icon"
import { getPageTitle } from "@/lib/nav"
import { cn } from "@/lib/utils"
import { useTabs } from "@/hooks/use-tabs"
import { TabBar } from "@/components/layout/tab-bar"
import { useMenuSearch } from "@/hooks/use-menu-search"

// 一级菜单项公共样式
const menuItemClass =
  "flex w-full items-center gap-3 overflow-hidden rounded-md p-2 text-left text-sm ring-sidebar-ring outline-hidden transition-[width,height,padding] hover:bg-sidebar-accent hover:text-sidebar-accent-foreground focus-visible:ring-2 active:bg-sidebar-accent active:text-sidebar-accent-foreground"

// 子菜单项公共样式
const subMenuItemClass =
  "flex h-7 min-w-0 -translate-x-px items-center gap-2 overflow-hidden rounded-md px-2 text-sm text-sidebar-foreground ring-sidebar-ring outline-hidden group-data-[collapsible=icon]:hidden hover:bg-sidebar-accent hover:text-sidebar-accent-foreground focus-visible:ring-2 active:bg-sidebar-accent active:text-sidebar-accent-foreground disabled:pointer-events-none disabled:opacity-50 aria-disabled:pointer-events-none aria-disabled:opacity-50 data-active:bg-sidebar-accent data-active:text-sidebar-accent-foreground [&>span:last-child]:truncate [&>svg]:size-4 [&>svg]:shrink-0 [&>svg]:text-sidebar-accent-foreground"

export function LayoutSidebar({ children }: { children: React.ReactNode }) {
  const pathname = usePathname()
  const { theme, setTheme } = useTheme()
  const { authAccount, getCurrentMenuTree } = useAuth()
  const {
    tabs,
    activePath,
    navigateToTab,
    closeTab,
    closeOtherTabs,
    closeAllTabs,
    refreshTab,
  } = useTabs()
  const accountData = {
    account: {
      name: authAccount?.name || "管理员",
      email: authAccount?.email || "",
    },
  }
  // 获取当前菜单树
  const menuTree = useMemo(() => getCurrentMenuTree(), [getCurrentMenuTree])
  const { searchKeyword, setSearchKeyword, searchResults, isSearching } =
    useMenuSearch(menuTree)

  // 用户手动切换的展开状态（仅记录显式操作）
  const [userOpenOverrides, setUserOpenOverrides] = useState<
    Record<string, boolean>
  >({})

  // 菜单展开状态：用户覆盖优先，否则根据 pathname 自动判断
  const menuOpenState = useMemo(() => {
    const autoState: Record<string, boolean> = {}
    menuTree.forEach((item) => {
      if (item.children && item.children.length > 0) {
        autoState[item.id.toString()] = item.children.some((child) =>
          pathname.startsWith(child.path || "")
        )
      }
    })
    return { ...autoState, ...userOpenOverrides }
  }, [menuTree, pathname, userOpenOverrides])

  const toggleMenu = (key: string) => {
    setUserOpenOverrides((prev) => ({
      ...prev,
      [key]: !menuOpenState[key],
    }))
  }

  return (
    <SidebarProvider>
      <Sidebar>
        <SidebarHeader>
          <div className="flex items-center justify-between px-4 py-2">
            <div className="flex items-center gap-2">
              <Building2 className="h-6 w-6" />
              <span className="text-lg font-semibold">管理后台</span>
            </div>
            <Button
              variant="ghost"
              size="icon"
              onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
              title={theme === "dark" ? "切换到亮色模式" : "切换到暗色模式"}
            >
              {theme === "dark" ? (
                <Sun className="h-5 w-5" />
              ) : (
                <Moon className="h-5 w-5" />
              )}
              <span className="sr-only">切换主题</span>
            </Button>
          </div>
          {/* 菜单搜索框 */}
          <div className="px-3 pb-2">
            <div className="relative">
              <Search className="pointer-events-none absolute top-1/2 left-2.5 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
              <Input
                value={searchKeyword}
                onChange={(e) => setSearchKeyword(e.target.value)}
                placeholder="搜索菜单..."
                className="h-8 pr-7 pl-8 text-sm"
              />
              {searchKeyword && (
                <button
                  type="button"
                  onClick={() => setSearchKeyword("")}
                  className="absolute top-1/2 right-2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                  aria-label="清除搜索"
                >
                  <X className="h-3.5 w-3.5" />
                </button>
              )}
            </div>
          </div>
        </SidebarHeader>
        <SidebarContent>
          {isSearching ? (
            // ==================== 搜索结果视图 ====================
            <SidebarGroup>
              <SidebarGroupContent>
                <div className="px-2">
                  <p className="px-2 py-1.5 text-xs text-muted-foreground">
                    {searchResults.length > 0
                      ? `找到 ${searchResults.length} 个结果`
                      : "无匹配结果"}
                  </p>
                  <div className="space-y-0.5">
                    {searchResults.map((item) => (
                      <Link
                        key={item.id}
                        href={item.path}
                        onClick={() => setSearchKeyword("")}
                        className={cn(
                          "flex items-center gap-2.5 rounded-md px-2 py-1.5 text-sm transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground",
                          pathname === item.path &&
                            "bg-sidebar-accent font-medium text-sidebar-accent-foreground"
                        )}
                      >
                        {item.icon && (
                          <Icon name={item.icon} className="h-4 w-4 shrink-0" />
                        )}
                        <span className="min-w-0 flex-1 truncate">
                          {item.name}
                        </span>
                        {item.parentName && (
                          <span className="shrink-0 text-xs text-muted-foreground">
                            {item.parentName}
                          </span>
                        )}
                      </Link>
                    ))}
                    {searchResults.length === 0 && (
                      <div className="px-2 py-8 text-center text-sm text-muted-foreground">
                        未找到匹配的菜单
                      </div>
                    )}
                  </div>
                </div>
              </SidebarGroupContent>
            </SidebarGroup>
          ) : (
            // ==================== 正常菜单树视图 ====================
            <SidebarGroup>
              <SidebarGroupContent>
                <SidebarMenu>
                  {menuTree.map((item: AuthMenuTree) => (
                    <SidebarMenuItem key={item.id}>
                      {!item.children || item.children.length === 0 ? (
                        item.path ? (
                          <Link
                            href={item.path}
                            className={cn(
                              menuItemClass,
                              pathname === item.path &&
                                "bg-sidebar-accent font-medium text-sidebar-accent-foreground"
                            )}
                          >
                            {item.icon && <Icon name={item.icon} />}
                            <span>{item.name}</span>
                          </Link>
                        ) : (
                          <div className={menuItemClass}>
                            {item.icon && <Icon name={item.icon} />}
                            <span>{item.name}</span>
                          </div>
                        )
                      ) : (
                        <>
                          <SidebarMenuButton
                            onClick={(e) => {
                              e.stopPropagation()
                              toggleMenu(item.id.toString())
                            }}
                            className="flex w-full items-center justify-between gap-3"
                            aria-expanded={menuOpenState[item.id.toString()]}
                            aria-label={`${menuOpenState[item.id.toString()] ? "折叠" : "展开"}${item.name}菜单`}
                          >
                            <div className="flex items-center gap-3">
                              {item.icon && <Icon name={item.icon} />}
                              <span>{item.name}</span>
                            </div>
                            {menuOpenState[item.id.toString()] ? (
                              <ChevronDown className="h-3 w-3" />
                            ) : (
                              <ChevronRight className="h-3 w-3" />
                            )}
                          </SidebarMenuButton>
                          {menuOpenState[item.id.toString()] &&
                            item.children && (
                              <SidebarMenuSub>
                                {item.children.map((subItem: AuthMenuTree) => (
                                  <SidebarMenuSubItem key={subItem.id}>
                                    {subItem.path ? (
                                      <Link
                                        href={subItem.path}
                                        className={cn(
                                          subMenuItemClass,
                                          pathname.startsWith(subItem.path) &&
                                            "bg-sidebar-accent text-sidebar-accent-foreground"
                                        )}
                                      >
                                        {subItem.icon && (
                                          <Icon name={subItem.icon} />
                                        )}
                                        {subItem.name}
                                      </Link>
                                    ) : (
                                      <div className={subMenuItemClass}>
                                        {subItem.icon && (
                                          <Icon name={subItem.icon} />
                                        )}
                                        {subItem.name}
                                      </div>
                                    )}
                                  </SidebarMenuSubItem>
                                ))}
                              </SidebarMenuSub>
                            )}
                        </>
                      )}
                    </SidebarMenuItem>
                  ))}
                </SidebarMenu>
              </SidebarGroupContent>
            </SidebarGroup>
          )}
        </SidebarContent>
        <SidebarFooter>
          <NavAccount account={accountData.account} />
        </SidebarFooter>
        <SidebarRail />
      </Sidebar>
      <SidebarInset className="min-w-0 overflow-hidden">
        <header className="flex h-14 shrink-0 items-center gap-2 border-b px-4">
          <SidebarTrigger className="-ml-1" />
          <h1 className="flex-1 truncate font-semibold">
            {getPageTitle(pathname)}
          </h1>
          {/* 移动端主题切换：侧边栏在移动端隐藏时仍可切换主题 */}
          <Button
            variant="ghost"
            size="icon"
            className="md:hidden"
            onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
            title={theme === "dark" ? "切换到亮色模式" : "切换到暗色模式"}
          >
            {theme === "dark" ? (
              <Sun className="h-5 w-5" />
            ) : (
              <Moon className="h-5 w-5" />
            )}
            <span className="sr-only">切换主题</span>
          </Button>
        </header>
        {/* 多标签页栏 */}
        <TabBar
          tabs={tabs}
          activePath={activePath}
          onNavigate={navigateToTab}
          onClose={closeTab}
          onCloseOthers={closeOtherTabs}
          onCloseAll={closeAllTabs}
          onRefresh={refreshTab}
        />
        <div className="min-w-0 flex-1 overflow-y-auto p-4">{children}</div>
      </SidebarInset>
    </SidebarProvider>
  )
}
