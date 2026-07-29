"use client"

import { AuthGuard } from "@/components/layout/auth-guard"
import { LayoutSidebar } from "@/components/layout/sidebar"

export default function AdminLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <div className="flex min-h-screen">
      <AuthGuard>
        <LayoutSidebar>{children}</LayoutSidebar>
      </AuthGuard>
    </div>
  )
}
