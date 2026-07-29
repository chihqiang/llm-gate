"use client"

import {
  AuthAccount,
  AuthMenuTree,
  getAccount,
  menuTree,
  removeAccount,
  setAccount,
} from "@/lib/account"
import {
  AuthToken,
  getToken,
  removeToken,
  setToken,
  emitLogout,
} from "@/lib/token"
import {
  createContext,
  useContext,
  useState,
  useEffect,
  useCallback,
  useMemo,
  ReactNode,
} from "react"

/**
 * Auth Context 状态类型
 */
interface AuthContextType {
  /** 当前账户认证令牌 */
  authToken: AuthToken | null
  /** 当前账户信息 */
  authAccount: AuthAccount | null
  /** 认证初始化加载状态 */
  isLoading: boolean
  /** 登录方法 */
  login: (data: AuthToken) => void
  /** 设置当前账户方法 */
  setCurrentAccount: (account: AuthAccount) => void
  /** 获取当前账户菜单树 */
  getCurrentMenuTree: () => AuthMenuTree[]
  /** 退出方法 */
  logout: () => void
  /** 获取当前登录账户ID */
  getID: () => number | null
  /** 判断是否登录 */
  isLoggedIn: () => boolean
}

/**
 * 创建 Auth Context
 */
const AuthContext = createContext<AuthContextType | undefined>(undefined)

/**
 * Auth Provider 组件属性
 */
interface AuthProviderProps {
  children: ReactNode
}

/**
 * Auth Provider 组件
 * 包裹在应用根组件，提供全局认证状态
 */
export const AuthProvider = ({ children }: AuthProviderProps) => {
  // 全局共享的认证状态
  const [authToken, setAuthToken] = useState<AuthToken | null>(null)
  const [authAccount, setAuthAccount] = useState<AuthAccount | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  // 初始化：从 localStorage 读取认证信息
  useEffect(() => {
    const initAuth = () => {
      try {
        const data = getToken()
        if (data) {
          setAuthToken(data)
        }
        const account = getAccount()
        if (account) {
          setAuthAccount(account)
        }
      } catch (e) {
        console.error("Auth 初始化错误：", e)
      } finally {
        setIsLoading(false)
      }
    }

    initAuth()
  }, [])

  // 登录方法
  const login = useCallback((data: AuthToken) => {
    setAuthToken(data)
    setToken(data)
  }, [])

  // 设置当前账户方法
  const setCurrentAccount = useCallback((account: AuthAccount) => {
    setAuthAccount(account)
    setAccount(account)
  }, [])

  // 获取当前账户菜单树
  const getCurrentMenuTree = useCallback(() => {
    const account = authAccount || getAccount()
    if (!account) {
      return []
    }
    return menuTree(account.menus)
  }, [authAccount])

  // 退出方法
  const logout = useCallback(() => {
    setAuthToken(null)
    setAuthAccount(null)
    removeToken()
    removeAccount()
    emitLogout("用户已登出")
  }, [])

  // 获取当前登录账户ID
  const getID = useCallback((): number | null => {
    return authToken?.id ?? null
  }, [authToken])

  // 判断是否登录
  const isLoggedIn = useCallback((): boolean => {
    return !!authToken
  }, [authToken])

  // Context 值
  const contextValue = useMemo<AuthContextType>(
    () => ({
      authToken,
      authAccount,
      isLoading,
      login,
      setCurrentAccount,
      getCurrentMenuTree,
      logout,
      getID,
      isLoggedIn,
    }),
    [
      authToken,
      authAccount,
      isLoading,
      login,
      setCurrentAccount,
      getCurrentMenuTree,
      logout,
      getID,
      isLoggedIn,
    ]
  )

  return (
    <AuthContext.Provider value={contextValue}>{children}</AuthContext.Provider>
  )
}

/**
 * 使用 Auth Context 的 Hook
 * 必须在 AuthProvider 内使用
 */
export const useAuthContext = (): AuthContextType => {
  const context = useContext(AuthContext)
  if (context === undefined) {
    throw new Error("useAuthContext 必须在 AuthProvider 内使用")
  }
  return context
}
