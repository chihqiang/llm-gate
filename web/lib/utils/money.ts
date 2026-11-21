/**
 * 金额工具：后端金额统一以「分」存储，展示统一转换为「元」。
 */

/** 将分转换为元字符串（保留两位小数），如 1234 -> "12.34" */
export function formatCents(cents: number | null | undefined): string {
  if (cents === null || cents === undefined) return "0.00"
  return (cents / 100).toFixed(2)
}

/** 带货币符号的金额展示，如 1234 -> "¥12.34" */
export function formatMoney(cents: number | null | undefined): string {
  return `¥${formatCents(cents)}`
}

/** 正负数带符号展示，如 1234 -> "+¥12.34"，-500 -> "-¥5.00" */
export function formatSignedMoney(cents: number): string {
  const prefix = cents > 0 ? "+" : ""
  return `${prefix}¥${formatCents(cents)}`
}
