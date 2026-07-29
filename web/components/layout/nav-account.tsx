"use client"

import { useEffect, useId, useRef, useState } from "react"
import { KeyIcon, LogOutIcon, MoreVerticalIcon } from "lucide-react"

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { SidebarMenu, SidebarMenuItem } from "@/components/ui/sidebar"
import { useAuth } from "@/hooks/use-auth"
import { ApiKeyDialog } from "@/components/forms/api-key-dialog"
import { toast } from "sonner"
import { useRouter } from "next/navigation"
import { getAvatarInitials } from "@/lib/utils/string"

export function NavAccount({
  account,
}: {
  account: {
    name: string
    email: string
    avatar?: string
  }
}) {
  const id = useId()
  const { logout, getID } = useAuth()
  const router = useRouter()
  const timeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const [keyDialogOpen, setKeyDialogOpen] = useState(false)
  const accountId = getID() ?? 0

  useEffect(() => {
    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current)
      }
    }
  }, [])

  const handleLogout = () => {
    logout()
    toast.success("退出登录成功", {
      description: "正在跳转至登录页...",
    })
    timeoutRef.current = setTimeout(() => {
      router.push("/login")
    }, 1000)
  }
  return (
    <>
      <SidebarMenu>
        <SidebarMenuItem>
          <DropdownMenu>
            <DropdownMenuTrigger id={`${id}-trigger`} suppressHydrationWarning>
              <div className="flex w-full items-center gap-2 overflow-hidden rounded-md px-3 py-2 text-sm transition-colors hover:bg-sidebar-accent hover:text-sidebar-accent-foreground data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground">
                <Avatar className="h-8 w-8 rounded-lg grayscale">
                  <AvatarImage src={account.avatar} alt={account.name} />
                  <AvatarFallback className="rounded-lg">
                    {getAvatarInitials(account.name)}
                  </AvatarFallback>
                </Avatar>
                <div className="grid flex-1 text-left text-sm leading-tight">
                  <span className="truncate font-medium">{account.name}</span>
                  <span className="truncate text-xs text-muted-foreground">
                    {account.email}
                  </span>
                </div>
                <MoreVerticalIcon className="ml-auto size-4" />
              </div>
            </DropdownMenuTrigger>
            <DropdownMenuContent
              className="w-[--radix-dropdown-menu-trigger-width] min-w-56 rounded-lg"
              align="end"
              sideOffset={4}
            >
              <DropdownMenuGroup>
                <DropdownMenuLabel className="p-0 font-normal">
                  <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                    <Avatar className="h-8 w-8 rounded-lg">
                      <AvatarImage src={account.avatar} alt={account.name} />
                      <AvatarFallback className="rounded-lg">
                        {getAvatarInitials(account.name)}
                      </AvatarFallback>
                    </Avatar>
                    <div className="grid flex-1 text-sm leading-tight">
                      <span className="truncate font-medium">
                        {account.name}
                      </span>
                      <span className="truncate text-xs text-muted-foreground">
                        {account.email}
                      </span>
                    </div>
                  </div>
                </DropdownMenuLabel>
              </DropdownMenuGroup>
              <DropdownMenuSeparator />
              <DropdownMenuGroup>
                <DropdownMenuItem onClick={() => setKeyDialogOpen(true)}>
                  <KeyIcon className="mr-2 h-4 w-4" />
                  API Key
                </DropdownMenuItem>
              </DropdownMenuGroup>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={handleLogout}>
                <LogOutIcon className="mr-2 h-4 w-4" />
                退出登录
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </SidebarMenuItem>
      </SidebarMenu>
      <ApiKeyDialog
        open={keyDialogOpen}
        onOpenChange={setKeyDialogOpen}
        accountId={accountId}
      />
    </>
  )
}
