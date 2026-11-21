import { IStorage } from "."

/**
 * localStorage 实现类
 * 实现 IStorage 接口，采用单例模式确保全局唯一
 */
class Local implements IStorage {
  private static instance: Local
  private constructor() {}

  public static getInstance(): Local {
    if (!Local.instance) {
      Local.instance = new Local()
    }
    return Local.instance
  }

  private get isBrowser(): boolean {
    return typeof window !== "undefined"
  }

  public set<T>(key: string, value: T): void {
    if (!this.isBrowser) return
    try {
      localStorage.setItem(key, JSON.stringify(value))
    } catch (err) {
      console.error(`[local] set error: ${key}`, err)
    }
  }

  public get<T>(key: string, defaultValue: T): T {
    if (!this.isBrowser) return defaultValue
    try {
      const data = localStorage.getItem(key)
      if (data === null) return defaultValue
      return JSON.parse(data) as T
    } catch (err) {
      console.error(`[local] get error: ${key}`, err)
      return defaultValue
    }
  }

  public remove(key: string): void {
    if (!this.isBrowser) return
    localStorage.removeItem(key)
  }

  public clear(): void {
    if (!this.isBrowser) return
    localStorage.clear()
  }

  public has(key: string): boolean {
    if (!this.isBrowser) return false
    return localStorage.getItem(key) !== null
  }

  public keys(): string[] {
    if (!this.isBrowser) return []
    return Object.keys(localStorage)
  }

  public size(): number {
    if (!this.isBrowser) return 0
    return localStorage.length
  }
}

/**
 * 创建并导出唯一的本地存储实例
 * 全局使用这一个实例即可
 */
export const local = Local.getInstance()
