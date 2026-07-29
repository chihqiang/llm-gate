"use client"

import * as React from "react"
import { useIsMobile } from "@/hooks/use-mobile"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Edit, Trash2 } from "lucide-react"
import { cn } from "@/lib/utils"

// -----------------------------------------------------------------------------
// 类型定义
// -----------------------------------------------------------------------------

/**
 * 表格列配置项
 * @template T 表格数据行类型
 */
export interface DataListColumn<T> {
  /** 列唯一标识 */
  key: string
  /** 表头显示文字 */
  header: string
  /** 表头自定义类名 */
  headerClassName?: string
  /** 单元格自定义类名 */
  cellClassName?: string
  /** 单元格渲染函数 */
  cell: (row: T) => React.ReactNode
}

/**
 * 分页控制配置
 */
export interface DataListPagination {
  /** 当前页码，从 1 开始 */
  page: number
  /** 每页条数 */
  pageSize: number
  /** 数据总条数 */
  total: number
  /** 页码切换回调 */
  onPageChange: (page: number) => void
}

/**
 * 通用列表组件 Props（支持桌面端表格 + 移动端卡片）
 * @template T 列表数据项类型
 */
interface DataListProps<T> {
  /** 列表数据数组 */
  data: T[]
  /** 表格列配置 */
  columns: DataListColumn<T>[]
  /** 移动端卡片渲染方法（不传则使用默认卡片，自动从 columns 渲染数据） */
  renderCard?: (row: T) => React.ReactNode
  /** 获取行唯一 key */
  keyExtractor: (row: T) => string
  /** 是否加载中 */
  loading?: boolean
  /** 分页配置（不传则不显示分页） */
  pagination?: DataListPagination
  /** 空数据提示文字 */
  emptyText?: string
  /** 外层自定义类名 */
  className?: string
  /** 是否可选择（显示复选框） */
  selectable?: boolean
  /** 选中的行 key 集合 */
  selectedRowKeys?: Set<string>
  /** 是否全选 */
  isAllSelected?: boolean
  /** 是否部分选中 */
  isPartiallySelected?: boolean
  /** 选中行变化回调 */
  onSelectRow?: (key: string, checked: boolean) => void
  /** 全选/取消全选回调 */
  onSelectAll?: (checked: boolean) => void
  /** 表格是否自带边框 @default true */
  bordered?: boolean
}

// -----------------------------------------------------------------------------
// 分页工具：生成页码数组（1...5 6 7...10 风格）
// -----------------------------------------------------------------------------

/**
 * 计算要显示的页码数组，支持省略号
 * @param currentPage 当前页
 * @param totalPages 总页数
 * @returns 如 [1, "...", 5,6,7, "...", 10]
 */
function generatePaginationNumbers(
  currentPage: number,
  totalPages: number
): (number | "...")[] {
  // 总页数 ≤7 时直接显示全部
  if (totalPages <= 7) {
    return Array.from({ length: totalPages }, (_, i) => i + 1)
  }

  const pages: (number | "...")[] = []

  /**
   * 添加一个页码，自动判断是否插入省略号
   */
  const addPageItem = (page: number) => {
    const lastItem = pages[pages.length - 1]
    // 如果上一个是数字且不连续，中间加省略号
    if (typeof lastItem === "number" && lastItem + 1 < page) {
      pages.push("...")
    }
    pages.push(page)
  }

  // 需要显示的关键页：第一页、最后一页、当前页前后各一页
  const keyPages = new Set<number>([
    1,
    totalPages,
    currentPage - 1,
    currentPage,
    currentPage + 1,
  ])

  // 过滤合法页码并排序后添加
  Array.from(keyPages)
    .filter((p) => p >= 1 && p <= totalPages)
    .sort((a, b) => a - b)
    .forEach(addPageItem)

  return pages
}

// -----------------------------------------------------------------------------
// 子组件：分页栏
// -----------------------------------------------------------------------------

/**
 * 列表分页组件
 */
function DataListPaginationBar({
  pagination,
}: {
  pagination: DataListPagination
}) {
  const { page, pageSize, total, onPageChange } = pagination
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  // 只有1页时不显示分页
  if (totalPages <= 1) return null

  const pageNumbers = generatePaginationNumbers(page, totalPages)

  return (
    <div className="flex flex-col items-center gap-2 sm:flex-row sm:justify-between">
      {/* 分页信息文字 */}
      <p className="text-sm text-muted-foreground">
        共 {total} 条 · 第 {page} / {totalPages} 页
      </p>

      {/* shadcn 分页组件 */}
      <Pagination className="w-auto">
        <PaginationContent>
          {/* 上一页 */}
          <PaginationItem>
            <PaginationPrevious
              href="#"
              text="上一页"
              onClick={(e) => {
                e.preventDefault()
                if (page > 1) onPageChange(page - 1)
              }}
              aria-disabled={page <= 1}
              className={cn(page <= 1 && "pointer-events-none opacity-40")}
            />
          </PaginationItem>

          {/* 数字页码 & 省略号 */}
          {pageNumbers.map((item, index) =>
            item === "..." ? (
              <PaginationItem key={`el-${index}-ellipsis`}>
                <PaginationEllipsis />
              </PaginationItem>
            ) : (
              <PaginationItem key={item}>
                <PaginationLink
                  href="#"
                  isActive={item === page}
                  onClick={(e) => {
                    e.preventDefault()
                    onPageChange(item)
                  }}
                >
                  {item}
                </PaginationLink>
              </PaginationItem>
            )
          )}

          {/* 下一页 */}
          <PaginationItem>
            <PaginationNext
              href="#"
              text="下一页"
              onClick={(e) => {
                e.preventDefault()
                if (page < totalPages) onPageChange(page + 1)
              }}
              aria-disabled={page >= totalPages}
              className={cn(
                page >= totalPages && "pointer-events-none opacity-40"
              )}
            />
          </PaginationItem>
        </PaginationContent>
      </Pagination>
    </div>
  )
}

// -----------------------------------------------------------------------------
// 主组件：DataList（自适应表格/卡片）
// -----------------------------------------------------------------------------

/**
 * 通用列表组件
 * 桌面：Table 布局
 * 移动端：Card 布局
 * 支持：加载、空状态、分页、自定义列
 */
/**
 * 默认移动端卡片：自动从 columns 配置渲染数据
 * 无需使用者传入 renderCard 即可自动生成移动端卡片视图
 */
function DefaultCard<T>({
  row,
  columns,
}: {
  row: T
  columns: DataListColumn<T>[]
}) {
  // 过滤掉操作列，避免在卡片数据展示中渲染按钮
  const dataColumns = columns.filter((col) => col.key !== "actions")
  return (
    <Card>
      <CardContent className="p-4">
        <dl className="grid grid-cols-1 gap-x-4 gap-y-2 text-sm sm:grid-cols-2">
          {dataColumns.map((col) => (
            <div key={col.key} className="flex items-start gap-2">
              <dt className="shrink-0 font-medium text-muted-foreground">
                {col.header}:
              </dt>
              <dd className="min-w-0 flex-1 break-words">{col.cell(row)}</dd>
            </div>
          ))}
        </dl>
      </CardContent>
    </Card>
  )
}

export function DataList<T>({
  data,
  columns,
  renderCard,
  keyExtractor,
  loading = false,
  pagination,
  emptyText = "暂无数据",
  className,
  selectable = false,
  selectedRowKeys,
  isAllSelected = false,
  isPartiallySelected = false,
  onSelectRow,
  onSelectAll,
  bordered = true,
}: DataListProps<T>) {
  const effectiveSelectedRowKeys = selectedRowKeys ?? new Set<string>()
  const isMobile = useIsMobile()
  const isEmpty = !loading && data.length === 0

  const checkboxColSpan = selectable ? 1 : 0
  const totalColumns = columns.length + checkboxColSpan

  // 渲染单张卡片（含选择框）
  const renderMobileCard = (row: T) => {
    const key = keyExtractor(row)
    const isSelected = effectiveSelectedRowKeys.has(key)
    const cardContent = renderCard ? (
      renderCard(row)
    ) : (
      <DefaultCard row={row} columns={columns} />
    )

    if (!selectable) return cardContent

    return (
      <div className="flex items-start gap-3">
        <div className="pt-4 pl-3">
          <Checkbox
            checked={isSelected}
            onCheckedChange={(checked) => onSelectRow?.(key, !!checked)}
          />
        </div>
        <div className="min-w-0 flex-1">{cardContent}</div>
      </div>
    )
  }

  return (
    <div className={cn("space-y-4", className)}>
      {/* ==================== 桌面端：表格视图 ==================== */}
      {!isMobile && (
        <div
          className={cn(
            "overflow-hidden",
            bordered && "rounded-lg border border-border/60"
          )}
        >
          <Table>
            {/* 表头 */}
            <TableHeader>
              <TableRow className="bg-muted/30 hover:bg-muted/30">
                {selectable && (
                  <TableHead className="w-12">
                    <Checkbox
                      checked={isAllSelected}
                      {...(isPartiallySelected
                        ? { "data-state": "indeterminate" as const }
                        : {})}
                      onCheckedChange={(checked) => onSelectAll?.(!!checked)}
                    />
                  </TableHead>
                )}
                {columns.map((col) => (
                  <TableHead
                    key={col.key}
                    className={cn(
                      "h-10 px-4 text-xs font-semibold tracking-wide text-muted-foreground uppercase",
                      col.headerClassName
                    )}
                  >
                    {col.header}
                  </TableHead>
                ))}
              </TableRow>
            </TableHeader>

            {/* 表格内容 */}
            <TableBody>
              {/* 加载中状态 */}
              {loading && (
                <TableRow className="hover:bg-transparent">
                  <TableCell
                    colSpan={totalColumns}
                    className="h-32 text-center text-muted-foreground"
                  >
                    <span className="inline-flex items-center gap-2">
                      <span className="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
                      加载中...
                    </span>
                  </TableCell>
                </TableRow>
              )}

              {/* 空数据状态 */}
              {!loading && isEmpty && (
                <TableRow className="hover:bg-transparent">
                  <TableCell
                    colSpan={totalColumns}
                    className="h-32 text-center text-muted-foreground"
                  >
                    {emptyText}
                  </TableCell>
                </TableRow>
              )}

              {/* 正常数据列表 */}
              {!loading &&
                !isEmpty &&
                data.map((row) => {
                  const key = keyExtractor(row)
                  const isSelected = effectiveSelectedRowKeys.has(key)
                  return (
                    <TableRow
                      key={key}
                      data-selected={isSelected || undefined}
                      className={cn(
                        isSelected && "bg-primary/8",
                        "transition-colors"
                      )}
                    >
                      {selectable && (
                        <TableCell className="px-4">
                          <Checkbox
                            checked={isSelected}
                            onCheckedChange={(checked) =>
                              onSelectRow?.(key, !!checked)
                            }
                          />
                        </TableCell>
                      )}
                      {columns.map((col) => (
                        <TableCell
                          key={col.key}
                          className={cn("px-4 py-2.5", col.cellClassName)}
                        >
                          {col.cell(row)}
                        </TableCell>
                      ))}
                    </TableRow>
                  )
                })}
            </TableBody>
          </Table>
          {/* 分页（嵌入表格底部） */}
          {pagination && (
            <div className="border-t px-4 py-2.5">
              <DataListPaginationBar pagination={pagination} />
            </div>
          )}
        </div>
      )}

      {/* ==================== 移动端：卡片视图 ==================== */}
      {isMobile && (
        <div className="space-y-3">
          {/* 全选栏（可选择时显示） */}
          {selectable && !loading && !isEmpty && (
            <div className="flex items-center gap-2 px-1">
              <Checkbox
                checked={isAllSelected}
                {...(isPartiallySelected
                  ? { "data-state": "indeterminate" as const }
                  : {})}
                onCheckedChange={(checked) => onSelectAll?.(!!checked)}
              />
              <span className="text-sm text-muted-foreground">
                全选 ({effectiveSelectedRowKeys.size}/{data.length})
              </span>
            </div>
          )}

          {/* 加载中 */}
          {loading && (
            <div className="flex h-32 items-center justify-center text-sm text-muted-foreground">
              加载中...
            </div>
          )}

          {/* 空数据 */}
          {!loading && isEmpty && (
            <div className="flex h-32 items-center justify-center text-sm text-muted-foreground">
              {emptyText}
            </div>
          )}

          {/* 卡片列表 */}
          {!loading &&
            !isEmpty &&
            data.map((row) => (
              <div key={keyExtractor(row)}>{renderMobileCard(row)}</div>
            ))}
        </div>
      )}

      {/* ==================== 移动端分页 ==================== */}
      {isMobile && pagination && (
        <DataListPaginationBar pagination={pagination} />
      )}
    </div>
  )
}

// -----------------------------------------------------------------------------
// 卡片组件：RenderCard（自定义卡片）
// 支持：标题、副标题、状态、元数据、操作按钮
// 可自定义图标
// 配合 DataList 组件使用，实现自定义卡片渲染
// -----------------------------------------------------------------------------
export interface RenderCardProps<T> {
  entity: T
  title: string
  subtitle?: string | React.ReactNode
  status?: {
    value: string
    variant: "default" | "destructive" | "secondary" | "outline"
    label: string
  }
  meta?: React.ReactNode
  onEdit?: (entity: T) => void
  onDelete?: (entity: T) => void
  icon?: React.ReactNode
}
// -----------------------------------------------------------------------------
// 卡片组件：RenderCard（自定义卡片）
// 支持：标题、副标题、状态、元数据、操作按钮
// 可自定义图标
// 配合 DataList 组件使用，实现自定义卡片渲染
// -----------------------------------------------------------------------------
export function RenderCard<T>({
  entity,
  title,
  subtitle,
  status,
  meta,
  onEdit,
  onDelete,
  icon,
}: RenderCardProps<T>) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            {icon}
            <span>{title}</span>
          </div>
          {status && <Badge variant={status.variant}>{status.label}</Badge>}
        </CardTitle>
        {subtitle && <CardDescription>{subtitle}</CardDescription>}
      </CardHeader>
      <CardContent className="flex items-center justify-between">
        <div className="flex items-center gap-4 text-sm text-muted-foreground">
          {meta}
        </div>
        <div className="flex items-center gap-2">
          {onEdit && (
            <Button variant="ghost" size="icon" onClick={() => onEdit(entity)}>
              <Edit className="h-4 w-4" />
            </Button>
          )}
          {onDelete && (
            <Button
              variant="ghost"
              size="icon"
              onClick={() => onDelete(entity)}
              className="text-destructive hover:text-destructive"
            >
              <Trash2 className="h-4 w-4" />
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  )
}
