"use client"

import { LoginForm } from "@/components/login/form"
import { useAuth } from "@/hooks/use-auth"
import { GalleryVerticalEnd } from "lucide-react"
import Image from "next/image"
import Link from "next/link"
import { useRouter } from "next/navigation"
import { useEffect } from "react"

export default function LoginPage() {
  const { isLoggedIn } = useAuth()
  const router = useRouter()
  // 检查是否已登录
  useEffect(() => {
    if (isLoggedIn()) {
      router.push("/admin")
    }
  }, [isLoggedIn, router])
  return (
    <div className="grid min-h-svh lg:grid-cols-2">
      <div className="flex flex-col gap-4 p-6 md:p-10">
        <div className="flex justify-center gap-2 md:justify-start">
          <Link
            href="#"
            className="flex items-center gap-2 font-medium"
            aria-label="首页"
          >
            <div className="flex size-6 items-center justify-center rounded-md bg-primary text-primary-foreground">
              <GalleryVerticalEnd className="size-4" />
            </div>
            Admin Inc.
          </Link>
        </div>
        <div className="flex flex-1 items-center justify-center">
          <div className="w-full max-w-xs">
            <LoginForm />
          </div>
        </div>
      </div>
      <div className="relative hidden bg-muted lg:block">
        <Image
          src="/static/image/login-left-bg.svg"
          alt="登录页面背景装饰图"
          fill
          loading="eager"
          className="object-cover dark:brightness-[0.2] dark:grayscale"
          sizes="100vw"
        />
      </div>
    </div>
  )
}
