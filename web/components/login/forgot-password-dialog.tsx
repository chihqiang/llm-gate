"use client"

import { useEffect, useRef, useState } from "react"
import { toast } from "sonner"
import { MultiStepDialog } from "@/components/widgets/multi-step-dialog"

interface ForgotPasswordDialogProps {
  className?: string
}

export function ForgotPasswordDialog({ className }: ForgotPasswordDialogProps) {
  const [countdown, setCountdown] = useState(0)
  const [, setEmail] = useState("")
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  useEffect(() => {
    return () => {
      if (timerRef.current) {
        clearInterval(timerRef.current)
      }
    }
  }, [])

  return (
    <MultiStepDialog
      className={className}
      triggerText="忘记密码？"
      steps={[
        {
          title: "找回密码",
          description: "请输入您的注册邮箱，我们将发送验证码给您。",
          fields: [
            {
              name: "email",
              label: "邮箱地址",
              type: "email",
              placeholder: "请输入注册邮箱",
              validate: (val) => {
                if (!val) return "请输入邮箱地址"
                if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(val))
                  return "请输入有效的邮箱地址"
                return null
              },
            },
          ],
          actionText: countdown > 0 ? `${countdown}s 后重发` : "发送验证码",
          onSubmitAction: async (values) => {
            setEmail(values.email ?? "")
            await new Promise((r) => setTimeout(r, 1000))
            toast.success("验证码已发送", {
              description: `验证码已发送至 ${values.email}，请查收。`,
            })
            setCountdown(60)
            if (timerRef.current) clearInterval(timerRef.current)
            timerRef.current = setInterval(() => {
              setCountdown((p) => {
                if (p <= 1) {
                  if (timerRef.current) clearInterval(timerRef.current)
                  timerRef.current = null
                  return 0
                }
                return p - 1
              })
            }, 1000)
          },
        },
        {
          title: "输入验证码",
          description: "请输入发送到您邮箱的6位验证码。",
          fields: [
            {
              name: "code",
              label: "验证码",
              placeholder: "请输入6位验证码",
              maxLength: 6,
              formatter: (v) => v.replace(/\D/g, ""),
              validate: (val) => (val.length !== 6 ? "请输入6位验证码" : null),
            },
          ],
          actionText: "验证",
        },
        {
          title: "设置新密码",
          description: "请设置您的新密码。",
          fields: [
            {
              name: "password",
              label: "新密码",
              type: "password",
              placeholder: "至少6位",
              validate: (val) =>
                val.length < 6 ? "密码长度不能少于6位" : null,
            },
            {
              name: "confirmPassword",
              label: "确认密码",
              type: "password",
              placeholder: "再次输入新密码",
              validate: (val, values) =>
                val !== values?.password ? "两次输入的密码不一致" : null,
            },
          ],
          actionText: "确认重置",
        },
      ]}
      onCompleteAction={async (values) => {
        await new Promise((r) => setTimeout(r, 1000))
        toast.success("密码重置成功", {
          description: "请使用新密码登录。",
        })
        if (timerRef.current) {
          clearInterval(timerRef.current)
          timerRef.current = null
        }
        setCountdown(0)
        setEmail("")
      }}
    />
  )
}
