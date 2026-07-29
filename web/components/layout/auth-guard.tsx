"use client"

import { useAuth } from "@/hooks/use-auth"
import { useEffect, useRef } from "react"
import { usePathname, useRouter } from "next/navigation"
import { authEventBus } from "@/lib/token"
import { hasMenuPath } from "@/lib/account"
import { Skeleton } from "@/components/ui/skeleton"

interface AuthGuardProps {
  children: React.ReactNode
}
export function AuthGuard({ children }: AuthGuardProps) {
  const { authToken, isLoading, authAccount } = useAuth()
  const router = useRouter()
  const pathname = usePathname()
  const redirectedRef = useRef(false)

  const isLoggedIn = !!authToken

  // 未登录 → 跳登录页
  useEffect(() => {
    if (!isLoading && !isLoggedIn) {
      router.push("/login")
    }
  }, [isLoading, isLoggedIn, router])

  // 监听未授权 / 登出事件
  useEffect(() => {
    const unsubscribeUnauthorized = authEventBus.on("auth:unauthorized", () => {
      router.push("/login")
    })
    const unsubscribeLogout = authEventBus.on("auth:logout", () => {
      router.push("/login")
    })
    return () => {
      unsubscribeUnauthorized()
      unsubscribeLogout()
    }
  }, [router])

  // 页面级权限检查：已登录后，检查当前路由是否在用户的菜单树中
  useEffect(() => {
    if (isLoading || !isLoggedIn || !authAccount) return
    if (redirectedRef.current) return

    // /admin/dashboard 始终允许访问
    if (pathname === "/admin/dashboard" || pathname === "/admin" || pathname === "/") {
      return
    }

    if (!hasMenuPath(pathname)) {
      redirectedRef.current = true
      router.replace("/admin/dashboard")
    }
  }, [isLoading, isLoggedIn, authAccount, pathname, router])

  if (isLoading) {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="space-y-3">
          <Skeleton className="h-8 w-48" />
          <Skeleton className="h-4 w-32" />
        </div>
      </div>
    )
  }

  if (!isLoggedIn) {
    return null
  }
  return <>{children}</>
}
