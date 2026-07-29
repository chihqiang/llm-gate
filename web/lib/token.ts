import { local } from "@/lib/storage"
import { eventBus } from "@/lib/events"

const TOKEN_KEY = "AUTH_TOKEN"

export interface AuthToken {
  id: number
  login_type?: string
  access_token: string
  token_type: string
  expires_in: number
  refresh_token?: string
}

/**
 * 认证事件数据接口
 */
export interface AuthEventData {
  message?: string
  timestamp?: number
}

/**
 * 认证事件总线
 * 是全局事件总线的引用，可用于监听所有认证相关事件
 */
export const authEventBus = eventBus

/**
 * 便捷函数：发出未授权事件
 */
export const emitUnauthorized = (message: string = "无效的授权凭证") => {
  authEventBus.emit("auth:unauthorized", {
    message,
    timestamp: Date.now(),
  } as AuthEventData)
}

/**
 * 便捷函数：发出登出事件
 */
export const emitLogout = (message: string = "已登出") => {
  authEventBus.emit("auth:logout", {
    message,
    timestamp: Date.now(),
  } as AuthEventData)
}

/**
 * 便捷函数：监听未授权事件
 */
export const onUnauthorized = (
  callback: (data: AuthEventData) => void
): (() => void) => {
  return authEventBus.on("auth:unauthorized", callback)
}

/**
 * 便捷函数：监听登出事件
 */
export const onLogout = (
  callback: (data: AuthEventData) => void
): (() => void) => {
  return authEventBus.on("auth:logout", callback)
}

/**
 * 获取本地认证信息
 */
export const getToken = (): AuthToken | null => {
  return local.get<AuthToken | null>(TOKEN_KEY, null)
}

/**
 * 保存认证信息
 */
export const setToken = (token: AuthToken) => {
  local.set(TOKEN_KEY, token)
}

/**
 * 删除 token（退出登录）
 */
export const removeToken = () => {
  local.remove(TOKEN_KEY)
}

/**
 * 获取 access_token
 * @returns token字符串，或null（未登录）
 */
export const getAccessToken = (): string | null => {
  const auth = getToken()
  if (!auth) return null
  return auth.access_token || null
}
