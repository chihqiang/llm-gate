"use client"
import { useEffect, useRef } from "react"
import { usePathname } from "next/navigation"
import NProgress from "nprogress"
import "nprogress/nprogress.css"

NProgress.configure({
  minimum: 0.3,
  easing: "ease",
  speed: 500,
  trickle: true,
  trickleSpeed: 200,
  showSpinner: false,
})

/**
 * 路由进度条 Provider 组件
 * 监听 pathname 变化，自动开始/结束进度条
 */
export function NProgressProvider() {
  const pathname = usePathname()
  const prevPathnameRef = useRef(pathname)
  const isLoadingRef = useRef(false)

  useEffect(() => {
    if (prevPathnameRef.current !== pathname) {
      NProgress.done()
      prevPathnameRef.current = pathname
      isLoadingRef.current = false
    }
  }, [pathname])

  useEffect(() => {
    const handleStart = () => {
      isLoadingRef.current = true
      NProgress.start()
    }
    const handleComplete = () => {
      NProgress.done()
      isLoadingRef.current = false
    }

    const handleAnchorClick = (e: MouseEvent) => {
      const anchor = (e.target as HTMLElement).closest("a")
      if (!anchor) return
      const href = anchor.getAttribute("href")
      if (
        !href ||
        href.startsWith("http") ||
        href.startsWith("#") ||
        anchor.target === "_blank"
      )
        return
      if (isLoadingRef.current) return
      handleStart()
    }

    document.addEventListener("click", handleAnchorClick)

    window.addEventListener("beforepopstate", handleStart)

    return () => {
      document.removeEventListener("click", handleAnchorClick)
      window.removeEventListener("beforepopstate", handleStart)
      NProgress.done()
    }
  }, [])

  return null
}
