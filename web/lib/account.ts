import { local } from "@/lib/storage"
import type { Menu } from "@/api/menu"

const ACCOUNT_KEY = "AUTH_ACCOUNT"

export interface AuthAccount {
  id: number
  name: string
  email: string
  status: boolean
  menus: Menu[]
}

export const setAccount = (account: AuthAccount) => {
  local.set(ACCOUNT_KEY, account)
}

export const removeAccount = () => {
  local.remove(ACCOUNT_KEY)
}

export const getAccount = (): AuthAccount | null => {
  return local.get<AuthAccount | null>(ACCOUNT_KEY, null)
}

export const hasMenuPath = (path: string): boolean => {
  return Boolean(getAccount()?.menus?.find((item: Menu) => item.path === path))
}

function matchApiUrl(pattern: string, url: string): boolean {
  const patternSegments = pattern.split("/")
  const urlSegments = url.split("/")

  function matchSegments(pi: number, ui: number): boolean {
    if (pi === patternSegments.length && ui === urlSegments.length) return true
    if (pi === patternSegments.length || ui === urlSegments.length) return false

    if (patternSegments[pi] === "**") {
      // ** 匹配零个或多个段
      for (let skip = ui; skip <= urlSegments.length; skip++) {
        if (matchSegments(pi + 1, skip)) return true
      }
      return false
    }

    if (patternSegments[pi] === "*") {
      return matchSegments(pi + 1, ui + 1)
    }

    if (patternSegments[pi] === urlSegments[ui]) {
      return matchSegments(pi + 1, ui + 1)
    }

    return false
  }

  return matchSegments(0, 0)
}

export const hasMenuApiUrl = (url: string): boolean => {
  return Boolean(
    getAccount()?.menus?.find((item: Menu) =>
      item.api_url ? matchApiUrl(item.api_url, url) : false
    )
  )
}

/**
 * 检查当前用户是否拥有指定 API 的访问权限（方法 + URL 联合检查）
 * @param method HTTP 方法（GET / POST / PUT / DELETE），传 "*" 忽略方法检查
 * @param url API 路径，如 "/api/v1/sys/accounts"
 * @returns 是否有权限
 */
export const hasMenuApiPermission = (method: string, url: string): boolean => {
  return Boolean(
    getAccount()?.menus?.find((item: Menu) => {
      if (!item.api_url) return false
      if (item.api_method !== "*" && item.api_method !== method) return false
      return matchApiUrl(item.api_url, url)
    })
  )
}

// 菜单树结构
export interface AuthMenuTree extends Menu {
  children: AuthMenuTree[]
}

// 构建菜单树，支持任意层级嵌套
export const menuTree = (menus: Menu[]): AuthMenuTree[] => {
  const nodeMap = new Map<number, AuthMenuTree>()
  const roots: AuthMenuTree[] = []

  menus.forEach((item) => {
    nodeMap.set(item.id, {
      ...item,
      children: [],
    })
  })

  menus.forEach((item) => {
    const node = nodeMap.get(item.id)!
    if (item.pid === 0) {
      roots.push(node)
      return
    }

    const parent = nodeMap.get(item.pid)
    if (parent) {
      parent.children.push(node)
    } else {
      roots.push(node)
    }
  })

  return roots
}
