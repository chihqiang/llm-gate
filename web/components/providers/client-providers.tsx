"use client"

import { useEffect } from "react"
import { usePathname } from "next/navigation"
import { AuthProvider } from "@/components/providers/auth-provider"
import { ThemeProvider } from "@/components/providers/theme-provider"
import { NProgressProvider } from "@/components/providers/nprogress-provider"
import { Toaster } from "sonner"
import { getPageTitle } from "@/lib/nav"

/**
 * 页面标题管理器
 * 根据当前路由路径自动设置浏览器标签页标题
 */
function PageTitleManager() {
  const pathname = usePathname()
  useEffect(() => {
    const pageTitle = getPageTitle(pathname)
    document.title = pageTitle ? `${pageTitle} - 管理后台` : "管理后台"
  }, [pathname])
  return null
}

/**
 * 客户端 Provider 聚合组件
 * 统一管理所有需要在客户端初始化的逻辑：
 * - MSW 模拟服务
 * - 页面标题
 * - 认证状态
 * - 主题切换
 * - 路由进度条
 * - Toast 通知
 */
export function ClientProviders({ children }: { children: React.ReactNode }) {
  return (
    <AuthProvider>
      <ThemeProvider>{children}</ThemeProvider>
      <Toaster />
      <NProgressProvider />
      <PageTitleManager />
    </AuthProvider>
  )
}
