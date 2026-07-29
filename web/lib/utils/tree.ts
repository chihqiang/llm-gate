/**
 * 树形选择节点基础结构
 */
export interface TreeSelectNode {
  id: number
  pid: number
  name: string
  children?: TreeSelectNode[]
  [key: string]: unknown
}

/**
 * 将扁平化的树节点数组转换为树形结构
 * @param data - 扁平化的节点数组，要求包含 id、pid 和 name 字段
 * @returns 树形结构的节点数组
 */
export function buildTree<T extends { id: number; pid: number; name: string }>(
  data: T[]
): TreeSelectNode[] {
  const nodeMap = new Map<number, TreeSelectNode>()
  const roots: TreeSelectNode[] = []

  data.forEach((item) => {
    nodeMap.set(item.id, { ...item, children: [] })
  })

  data.forEach((item) => {
    if (item.pid === 0) {
      roots.push(nodeMap.get(item.id)!)
    } else {
      const parent = nodeMap.get(item.pid)
      if (parent) {
        parent.children = parent.children || []
        parent.children.push(nodeMap.get(item.id)!)
      }
    }
  })

  return roots
}
