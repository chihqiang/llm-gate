"use client"

import { useEffect } from "react"
import { useRouter } from "next/navigation"

export default function LLMPage() {
  const router = useRouter()
  useEffect(() => {
    router.replace("/admin/sys/llm/providers")
  }, [router])
  return null
}
