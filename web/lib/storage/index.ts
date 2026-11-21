/**
 * 存储操作接口定义
 * 定义本地存储必须实现的标准方法，用于约束类的行为
 */
export interface IStorage {
  /**
   * 存储数据
   * @param key 存储的键名
   * @param value 存储的值（任意可序列化类型）
   */
  set<T>(key: string, value: T): void

  /**
   * 获取数据
   * @param key 存储的键名
   * @param defaultValue 取不到数据/解析失败时返回的默认值
   * @returns 对应类型的数据
   */
  get<T>(key: string, defaultValue: T): T

  /**
   * 删除指定键的数据
   * @param key 要删除的键名
   */
  remove(key: string): void

  /**
   * 清空所有 localStorage 数据
   */
  clear(): void

  /**
   * 判断某个键是否存在
   * @param key 键名
   * @returns 存在返回 true，不存在返回 false
   */
  has(key: string): boolean

  /**
   * 获取所有存储的键名数组
   * @returns 所有键名
   */
  keys(): string[]

  /**
   * 获取当前存储的数据总数
   * @returns 数量
   */
  size(): number
}

export { local } from "@/lib/storage/local"
