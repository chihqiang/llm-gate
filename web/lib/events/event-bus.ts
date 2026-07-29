/**
 * 通用事件总线系统
 * 支持任意类型的事件，可用于全局状态管理、事件通信等场景
 *
 * 使用示例：
 * ```typescript
 * // 监听事件
 * const unsubscribe = eventBus.on('user:login', (data) => {
 *   console.log('User logged in:', data)
 * })
 *
 * // 发出事件
 * eventBus.emit('user:login', { userId: 123, name: 'John' })
 *
 * // 取消监听
 * unsubscribe()
 * ```
 */

type EventListener<T = any> = (data: T) => void | Promise<void>

interface ListenerEntry<T = any> {
  callback: EventListener<T>
  once: boolean
}

/**
 * 通用事件总线
 * 支持事件监听、发出、单次监听等功能
 */
class EventBus {
  private listeners: Map<string, Set<ListenerEntry>> = new Map()

  /**
   * 监听事件
   * @param eventName 事件名称
   * @param callback 监听回调函数
   * @returns 取消监听函数
   */
  on<T = any>(eventName: string, callback: EventListener<T>): () => void {
    if (!this.listeners.has(eventName)) {
      this.listeners.set(eventName, new Set())
    }

    const entry: ListenerEntry<T> = { callback, once: false }
    this.listeners.get(eventName)!.add(entry)

    // 返回取消监听函数
    return () => {
      const set = this.listeners.get(eventName)
      if (set) {
        set.delete(entry)
      }
    }
  }

  /**
   * 单次监听事件（监听完成后自动取消）
   * @param eventName 事件名称
   * @param callback 监听回调函数
   * @returns 取消监听函数
   */
  once<T = any>(eventName: string, callback: EventListener<T>): () => void {
    if (!this.listeners.has(eventName)) {
      this.listeners.set(eventName, new Set())
    }

    const entry: ListenerEntry<T> = { callback, once: true }
    this.listeners.get(eventName)!.add(entry)

    // 返回取消监听函数
    return () => {
      const set = this.listeners.get(eventName)
      if (set) {
        set.delete(entry)
      }
    }
  }

  /**
   * 发出事件
   * @param eventName 事件名称
   * @param data 事件数据
   */
  async emit<T = any>(eventName: string, data?: T): Promise<void> {
    const set = this.listeners.get(eventName)
    if (!set || set.size === 0) return

    // 收集所有的Promise，以便处理异步回调
    const promises: Promise<void>[] = []

    // 快照遍历，防止回调中注册新监听器导致重入
    const entries = [...set]
    const toDelete: ListenerEntry[] = []

    for (const entry of entries) {
      try {
        const result = entry.callback(data)

        if (result instanceof Promise) {
          promises.push(result.catch(() => {}))
        }

        if (entry.once) {
          toDelete.push(entry)
        }
      } catch (error) {
        console.error(`Error in event listener for "${eventName}":`, error)
      }
    }

    toDelete.forEach((entry) => set.delete(entry))

    // 等待所有异步回调完成
    if (promises.length > 0) {
      await Promise.all(promises)
    }
  }

  /**
   * 移除特定事件的所有监听器
   * @param eventName 事件名称（不传则清空所有）
   */
  off(eventName?: string): void {
    if (eventName) {
      this.listeners.delete(eventName)
    } else {
      this.listeners.clear()
    }
  }

  /**
   * 获取指定事件的监听器数量
   * @param eventName 事件名称
   */
  getListenerCount(eventName: string): number {
    return this.listeners.get(eventName)?.size ?? 0
  }

  /**
   * 获取所有已注册的事件名称
   */
  getEventNames(): string[] {
    return Array.from(this.listeners.keys())
  }

  /**
   * 获取所有监听器总数
   */
  getTotalListenerCount(): number {
    let total = 0
    this.listeners.forEach((set) => {
      total += set.size
    })
    return total
  }
}

export const eventBus = new EventBus()
