/**
 * 获取头像显示的名字缩写
 * 英文：取前两部分首字母 John Doe → JD
 * 中文：取前两个字符 张三 → 张三
 * 空值返回 UN
 */
export const getAvatarInitials = (name?: string | null): string => {
  if (!name || !name.trim()) return "UN"
  const trimmed = name.trim()
  // 英文/拉丁姓名：取前两部分首字母（支持重音字符）
  if (/^[\p{L}\s]+$/u.test(trimmed)) {
    const parts = trimmed.split(/\s+/).filter(Boolean)
    if (parts.length >= 2) {
      return (parts[0][0] + parts[1][0]).toUpperCase()
    }
    return trimmed.substring(0, 2).toUpperCase()
  }
  // 中文/其他字符：直接取前两位
  return trimmed.substring(0, 2)
}
