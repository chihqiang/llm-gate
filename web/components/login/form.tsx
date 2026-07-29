import { cn } from "@/lib/utils"
import { Label } from "@/components/ui/label"
import { Input } from "@/components/ui/input"
import { Checkbox } from "@/components/ui/checkbox"
import { Button } from "@/components/ui/button"
import { toast } from "sonner"
import { loginApi, LoginRequest, userProfileApi } from "@/api/account"
import { useAuth } from "@/hooks/use-auth"
import { ForgotPasswordDialog } from "@/components/login/forgot-password-dialog"

import { useForm } from "@tanstack/react-form"
import { z } from "zod"
import { useRouter } from "next/navigation"
import { useCallback } from "react"

export function LoginForm({
  className,
  ...props
}: React.ComponentProps<"form">) {
  const router = useRouter()
  const { login, setCurrentAccount } = useAuth()

  const handleSubmit = useCallback(
    async (value: LoginRequest) => {
      try {
        // 登录
        const loginResp = await loginApi(value)
        login(loginResp)
        // 获取用户信息
        const accountResp = await userProfileApi()
        setCurrentAccount(accountResp)

        toast.success("登录成功", {
          description: "正在跳转至控制面板...",
        })
        setTimeout(() => {
          router.push("/")
        }, 1000)
      } catch (error: unknown) {
        const msg = error instanceof Error ? error.message : "登录失败"
        toast.error(msg, {
          description: "发生错误，请稍后重试。",
        })
      }
    },
    [login, setCurrentAccount, router]
  )

  const form = useForm({
    defaultValues: {
      email: "",
      password: "",
      remember: true,
    } as LoginRequest,
    validators: {
      onSubmit: z.object({
        email: z.string().min(1, "请输入邮箱"),
        password: z.string().min(1, "请输入密码"),
        remember: z.boolean(),
      }),
    },
    onSubmit: async ({ value }) => {
      await handleSubmit(value)
    },
  })

  return (
    <form
      className={cn("flex flex-col gap-6", className)}
      {...props}
      onSubmit={(e) => {
        e.preventDefault()
        form.handleSubmit()
      }}
    >
      <div className="flex flex-col items-center gap-2 text-center">
        <h1 className="text-2xl font-bold">登录您的账户</h1>
        <p className="text-sm text-balance text-muted-foreground">
          输入您的邮箱和密码进行登录
        </p>
      </div>

      <div className="grid gap-6">
        {/* 邮箱 */}
        <form.Field name="email">
          {(field) => (
            <div className="grid gap-3">
              <Label htmlFor="email">邮箱</Label>
              <Input
                id="email"
                value={field.state.value}
                onChange={(e) => {
                  field.handleChange(e.target.value)
                }}
                onBlur={() => {
                  field.handleBlur()
                }}
                placeholder="请输入邮箱"
              />
              {field.state.meta.errors?.[0]?.message && (
                <p className="text-sm text-red-500">
                  {field.state.meta.errors[0].message}
                </p>
              )}
            </div>
          )}
        </form.Field>

        {/* 密码 */}
        <form.Field name="password">
          {(field) => (
            <div className="grid gap-3">
              <div className="flex items-center justify-between">
                <Label htmlFor="password">密码</Label>
                <ForgotPasswordDialog />
              </div>
              <Input
                id="password"
                type="password"
                value={field.state.value}
                onChange={(e) => {
                  field.handleChange(e.target.value)
                }}
                onBlur={() => {
                  field.handleBlur()
                }}
                placeholder="请输入密码"
              />
              {field.state.meta.errors?.[0]?.message && (
                <p className="text-sm text-red-500">
                  {field.state.meta.errors[0].message}
                </p>
              )}
            </div>
          )}
        </form.Field>

        {/* 记住我 */}
        <form.Field name="remember">
          {(field) => (
            <div className="flex items-center space-x-2">
              <Checkbox
                id="remember"
                checked={field.state.value}
                onCheckedChange={(checked) => {
                  field.handleChange(Boolean(checked))
                }}
              />
              <label
                htmlFor="remember"
                className="text-sm leading-none font-medium"
              >
                30天内自动登录
              </label>
            </div>
          )}
        </form.Field>

        <Button
          type="submit"
          className="w-full"
          disabled={form.state.isSubmitting}
        >
          {form.state.isSubmitting ? "登录中..." : "登录"}
        </Button>
      </div>
    </form>
  )
}
