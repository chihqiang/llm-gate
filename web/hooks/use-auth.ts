"use client"

import { AuthAccount, AuthMenuTree } from "@/lib/account"
import { AuthToken } from "@/lib/token"
import { useAuthContext } from "@/components/providers/auth-provider"

/**
 * useAuth Hook 的返回类型
 */
export interface UseAuthReturn {
  /** 当前账户认证令牌 */
  authToken: AuthToken | null
  /** 当前账户信息（包含账户ID、姓名、邮箱、状态、菜单等） */
  authAccount: AuthAccount | null
  /** 认证初始化加载状态（用于控制加载中提示、避免权限判断异常） */
  isLoading: boolean
  /** 登录方法（传入 Token 完成登录） */
  login: (data: AuthToken) => void
  /** 设置当前账户方法（更新状态和本地存储） */
  setCurrentAccount: (account: AuthAccount) => void
  /** 获取当前账户菜单树（包含所有菜单、子菜单等） */
  getCurrentMenuTree: () => AuthMenuTree[]
  /** 退出方法（清空认证信息） */
  logout: () => void
  /** 获取当前登录账户ID（自动校验 Token 有效性） */
  getID: () => number | null
  /** 判断是否登录（核心权限判断） */
  isLoggedIn: () => boolean
}

/**
 * 权限管理自定义 Hooks（客户端专用）
 * 从全局 Auth Context 获取认证状态，确保状态全局同步
 * 封装账户登录、退出、Token 校验、账户信息获取等核心权限相关逻辑
 * 统一管理认证状态，供全局组件复用，避免重复开发
 * @returns 权限相关状态和方法集合
 */
export const useAuth = (): UseAuthReturn => {
  return useAuthContext()
}
