"use client"

import { useMemo, useState } from "react"
import type { AuthMenuTree } from "@/lib/account"

export interface FlatMenuItem {
  id: number
  name: string
  path: string
  icon: string
  parentName?: string
}

/**
 * 菜单搜索 Hook
 *
 * 将树形菜单扁平化为可搜索列表，
 * 根据关键词进行模糊匹配（名称 + 路径 + 父级名称）
 */
export function useMenuSearch(menuTree: AuthMenuTree[]) {
  const [searchKeyword, setSearchKeyword] = useState("")

  // 扁平化菜单树（只保留有 path 的叶子节点）
  const flatMenus = useMemo<FlatMenuItem[]>(() => {
    const result: FlatMenuItem[] = []

    const traverse = (items: AuthMenuTree[], parentName?: string) => {
      items.forEach((item) => {
        if (item.path) {
          result.push({
            id: item.id,
            name: item.name,
            path: item.path,
            icon: item.icon,
            parentName,
          })
        }
        if (item.children && item.children.length > 0) {
          traverse(item.children, item.name)
        }
      })
    }

    traverse(menuTree)
    return result
  }, [menuTree])

  // 搜索结果
  const searchResults = useMemo<FlatMenuItem[]>(() => {
    const keyword = searchKeyword.trim().toLowerCase()
    if (!keyword) return []

    return flatMenus.filter((item) => {
      return (
        item.name.toLowerCase().includes(keyword) ||
        item.path.toLowerCase().includes(keyword) ||
        (item.parentName?.toLowerCase().includes(keyword) ?? false)
      )
    })
  }, [flatMenus, searchKeyword])

  const isSearching = searchKeyword.trim().length > 0

  return {
    searchKeyword,
    setSearchKeyword,
    searchResults,
    isSearching,
  }
}
