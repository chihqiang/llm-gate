"use client"

import { useMemo, useState } from "react"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { TreeSelectNode } from "@/lib/utils/tree"

/**
 * 树形多选组件属性
 * @interface TreeCheckboxProps
 * @property data - 树形结构数据
 * @property selectedIds - 当前选中的节点ID列表
 * @property onChange - 选中状态变化时的回调函数
 * @property searchable - 是否显示搜索框，默认 true
 */
interface TreeCheckboxProps {
  data: TreeSelectNode[]
  selectedIds: number[]
  onChange: (selectedIds: number[]) => void
  searchable?: boolean
}

/**
 * 树形选择项组件属性
 */
interface TreeCheckboxItemProps {
  node: TreeSelectNode
  selectedIds: number[]
  onChange: (selectedIds: number[]) => void
  depth: number
  collapsedIds: Set<number>
  onToggleCollapse: (nodeId: number) => void
}

const descendantIdsCache = new WeakMap<TreeSelectNode, number[]>()

function getAllDescendantIds(node: TreeSelectNode): number[] {
  const cached = descendantIdsCache.get(node)
  if (cached) {
    return cached
  }

  const ids: number[] = []
  const traverse = (n: TreeSelectNode) => {
    ;(n.children || []).forEach((child) => {
      ids.push(child.id)
      traverse(child)
    })
  }
  traverse(node)

  descendantIdsCache.set(node, ids)
  return ids
}

/**
 * 获取所有节点ID列表
 */
function getAllNodeIds(nodes: TreeSelectNode[]): number[] {
  const ids: number[] = []
  const traverse = (n: TreeSelectNode) => {
    ids.push(n.id)
    ;(n.children || []).forEach((child) => traverse(child))
  }
  nodes.forEach((node) => traverse(node))
  return ids
}

/**
 * 检查节点名称是否匹配搜索关键词
 */
function isNodeMatch(node: TreeSelectNode, keyword: string): boolean {
  return node.name.toLowerCase().includes(keyword.toLowerCase())
}

/**
 * 检查节点及其后代是否匹配搜索关键词
 */
function isNodeOrDescendantMatch(
  node: TreeSelectNode,
  keyword: string
): boolean {
  if (isNodeMatch(node, keyword)) {
    return true
  }
  return (node.children || []).some((child) =>
    isNodeOrDescendantMatch(child, keyword)
  )
}

/**
 * 过滤树节点，保留匹配项及其祖先节点
 */
function filterTree(
  nodes: TreeSelectNode[],
  keyword: string
): TreeSelectNode[] {
  if (!keyword.trim()) {
    return nodes
  }

  return nodes.reduce<TreeSelectNode[]>((result, node) => {
    const match = isNodeOrDescendantMatch(node, keyword)

    if (match) {
      const filteredChildren = filterTree(node.children || [], keyword)

      const filteredNode: TreeSelectNode = {
        ...node,
        children:
          filteredChildren.length > 0 ? filteredChildren : node.children,
      }

      result.push(filteredNode)
    }

    return result
  }, [])
}

/**
 * 切换节点选中状态
 * - 选中时：同时选中节点及其所有后代
 * - 取消选中时：同时取消节点及其所有后代
 */
function TreeCheckboxItem({
  node,
  selectedIds,
  onChange,
  depth,
  collapsedIds,
  onToggleCollapse,
}: TreeCheckboxItemProps) {
  const selectedSet = useMemo(() => new Set(selectedIds), [selectedIds])
  const descendantIds = useMemo(() => getAllDescendantIds(node), [node])
  const hasChildren = (n: TreeSelectNode): boolean =>
    (n.children || []).length > 0
  const isCollapsed = collapsedIds.has(node.id)

  const toggleNode = (targetNode: TreeSelectNode) => {
    const descendantIds = getAllDescendantIds(targetNode)
    const isNodeSelected = selectedSet.has(targetNode.id)

    if (isNodeSelected) {
      onChange(
        selectedIds.filter(
          (id) => id !== targetNode.id && !descendantIds.includes(id)
        )
      )
    } else {
      const existingIds = new Set(selectedIds)
      const newIds = [targetNode.id]

      descendantIds.forEach((id) => {
        if (!existingIds.has(id)) {
          newIds.push(id)
        }
      })

      onChange([...selectedIds, ...newIds])
    }
  }

  /**
   * 判断节点是否被选中
   * 直接检查节点ID是否在选中集合中
   */
  const isChecked = useMemo(
    () => selectedSet.has(node.id),
    [selectedSet, node.id]
  )

  /**
   * 判断节点是否处于半选中状态
   * 当节点有子节点，且部分子节点被选中时显示半选中状态
   */
  const isIndeterminate = useMemo(() => {
    if ((node.children || []).length === 0) {
      return false
    }
    const selectedCount = descendantIds.filter((id) =>
      selectedSet.has(id)
    ).length
    if (selectedCount === 0) return false
    if (selectedCount === descendantIds.length) return false
    return true
  }, [descendantIds, selectedSet, node.children])

  return (
    <div key={node.id}>
      <label
        className="flex cursor-pointer items-center gap-2 rounded p-1 text-sm hover:bg-accent"
        style={{ paddingLeft: depth * 20 }}
      >
        <span className="w-4 shrink-0 text-center text-xs">
          {hasChildren(node) ? (
            <button
              onClick={(e) => {
                e.preventDefault()
                onToggleCollapse(node.id)
              }}
              className="h-4 w-4"
            >
              {isCollapsed ? "+" : "-"}
            </button>
          ) : null}
        </span>
        <Checkbox
          checked={isChecked}
          onCheckedChange={() => toggleNode(node)}
          {...(isIndeterminate ? { indeterminate: true } : {})}
        />
        <span className={hasChildren(node) ? "font-medium" : ""}>
          {node.name}
        </span>
      </label>
      {hasChildren(node) && !isCollapsed && (
        <div>
          {(node.children || []).map((child) => (
            <TreeCheckboxItem
              key={child.id}
              node={child}
              selectedIds={selectedIds}
              onChange={onChange}
              depth={depth + 1}
              collapsedIds={collapsedIds}
              onToggleCollapse={onToggleCollapse}
            />
          ))}
        </div>
      )}
    </div>
  )
}

/**
 * 树形多选组件
 * 支持无限层级树形结构的多选
 * - 选中父节点时自动选中所有子节点
 * - 取消父节点时自动取消所有子节点
 * - 部分选中时显示半选中状态
 * - 支持节点名称模糊搜索（当 searchable 为 true 时）
 * - 支持展开/收缩节点
 */
function TreeCheckbox({
  data,
  selectedIds,
  onChange,
  searchable = true,
}: TreeCheckboxProps) {
  const [searchValue, setSearchValue] = useState("")
  const [collapsedIds, setCollapsedIds] = useState<Set<number>>(new Set())

  const filteredData = useMemo(
    () => (searchable ? filterTree(data, searchValue) : data),
    [data, searchValue, searchable]
  )

  const allNodeIds = useMemo(() => getAllNodeIds(data), [data])

  const handleCollapseAll = () => {
    setCollapsedIds(new Set(allNodeIds))
  }

  const handleExpandAll = () => {
    setCollapsedIds(new Set())
  }

  const handleToggleCollapse = (nodeId: number) => {
    setCollapsedIds((prev) => {
      const newSet = new Set(prev)
      if (newSet.has(nodeId)) {
        newSet.delete(nodeId)
      } else {
        newSet.add(nodeId)
      }
      return newSet
    })
  }

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-center gap-2">
        {searchable && (
          <Input
            placeholder="搜索节点名称..."
            value={searchValue}
            onChange={(e) => setSearchValue(e.target.value)}
            className="min-w-0 flex-1"
          />
        )}
        <Button size="sm" variant="outline" onClick={handleExpandAll}>
          展开
        </Button>
        <Button size="sm" variant="outline" onClick={handleCollapseAll}>
          收缩
        </Button>
      </div>
      <div className="max-h-96 overflow-y-auto rounded-md border p-3">
        {filteredData.length > 0 ? (
          filteredData.map((node) => (
            <TreeCheckboxItem
              key={node.id}
              node={node}
              selectedIds={selectedIds}
              onChange={onChange}
              depth={0}
              collapsedIds={collapsedIds}
              onToggleCollapse={handleToggleCollapse}
            />
          ))
        ) : (
          <p className="text-sm text-muted-foreground">未找到匹配的节点</p>
        )}
      </div>
    </div>
  )
}

export { TreeCheckbox }
