"use client"

import {
  Home,
  User,
  Settings,
  Users,
  Shield,
  ShieldUser,
  Menu,
  FileText,
  LayoutDashboard,
  UserCog,
  ScrollText,
  Activity,
  ChevronDown,
  ChevronRight,
  Search,
  Plus,
  Edit,
  Trash2,
  Eye,
  EyeOff,
  Check,
  X,
  LogOut,
  Moon,
  Sun,
  Bell,
  Mail,
  Lock,
  Unlock,
  Key,
  RefreshCw,
  Download,
  Upload,
  MoreHorizontal,
  ArrowLeft,
  ArrowRight,
  Calendar,
  Clock,
  MapPin,
  Phone,
  Globe,
  Database,
  Server,
  Cpu,
  HardDrive,
  Wifi,
  WifiOff,
  Zap,
  AlertTriangle,
  AlertCircle,
  Info,
  HelpCircle,
  ExternalLink,
  Copy,
  Clipboard,
  Save,
  Filter,
  SortAsc,
  SortDesc,
  Columns,
  LayoutGrid,
  List,
  Table,
  Image,
  File,
  Folder,
  FolderOpen,
  Archive,
  Inbox,
  Send,
  MessageSquare,
  MessageCircle,
  Hash,
  AtSign,
  Percent,
  DollarSign,
  CreditCard,
  ShoppingCart,
  ShoppingBag,
  Package,
  Truck,
  Map,
  Navigation,
  Compass,
  Flag,
  Bookmark,
  Tag,
  Layers,
  Box,
  Triangle,
  Circle,
  Square,
  Hexagon,
  Octagon,
  Star,
  Heart,
  ThumbsUp,
  ThumbsDown,
  Hand,
  Pointer,
  Grip,
  Move,
  Maximize,
  Minimize,
  ZoomIn,
  ZoomOut,
  RotateCcw,
  RotateCw,
  Power,
  Plug,
  Battery,
  Terminal,
  Code,
  GitBranch,
  GitCommit,
  GitMerge,
  GitPullRequest,
  ExternalLinkIcon,
} from "lucide-react"
import type { LucideIcon } from "lucide-react"
import Link from "next/link"
import { useEffect, useRef, useState } from "react"
import { Input } from "@/components/ui/input"

const iconMap: Record<string, LucideIcon> = {
  Home,
  User,
  Settings,
  Users,
  Shield,
  ShieldUser,
  Menu,
  FileText,
  LayoutDashboard,
  UserCog,
  ScrollText,
  Activity,
  ChevronDown,
  ChevronRight,
  Search,
  Plus,
  Edit,
  Trash2,
  Eye,
  EyeOff,
  Check,
  X,
  LogOut,
  Moon,
  Sun,
  Bell,
  Mail,
  Lock,
  Unlock,
  Key,
  RefreshCw,
  Download,
  Upload,
  MoreHorizontal,
  ArrowLeft,
  ArrowRight,
  Calendar,
  Clock,
  MapPin,
  Phone,
  Globe,
  Database,
  Server,
  Cpu,
  HardDrive,
  Wifi,
  WifiOff,
  Zap,
  AlertTriangle,
  AlertCircle,
  Info,
  HelpCircle,
  ExternalLink,
  Copy,
  Clipboard,
  Save,
  Filter,
  SortAsc,
  SortDesc,
  Columns,
  LayoutGrid,
  List,
  Table,
  Image,
  File,
  Folder,
  FolderOpen,
  Archive,
  Inbox,
  Send,
  MessageSquare,
  MessageCircle,
  Hash,
  AtSign,
  Percent,
  DollarSign,
  CreditCard,
  ShoppingCart,
  ShoppingBag,
  Package,
  Truck,
  Map,
  Navigation,
  Compass,
  Flag,
  Bookmark,
  Tag,
  Layers,
  Box,
  Triangle,
  Circle,
  Square,
  Hexagon,
  Octagon,
  Star,
  Heart,
  ThumbsUp,
  ThumbsDown,
  Hand,
  Pointer,
  Grip,
  Move,
  Maximize,
  Minimize,
  ZoomIn,
  ZoomOut,
  RotateCcw,
  RotateCw,
  Power,
  Plug,
  Battery,
  Terminal,
  Code,
  GitBranch,
  GitCommit,
  GitMerge,
  GitPullRequest,
}

interface IconProps {
  name: string
  className?: string
}

export function Icon({ name, className = "h-4 w-4" }: IconProps) {
  const IconComponent = iconMap[name]
  if (!IconComponent) {
    if (process.env.NODE_ENV === "development") {
      console.warn(
        `[Icon] 未找到图标: "${name}"，可用图标: ${Object.keys(iconMap).join(", ")}`
      )
    }
    return null
  }
  return <IconComponent className={className} />
}

interface IconSelectorProps {
  value: string
  onChange: (value: string) => void
}

export function IconSelector({ value, onChange }: IconSelectorProps) {
  const [showHelp, setShowHelp] = useState(false)
  const helpRef = useRef<HTMLDivElement>(null)
  const triggerRef = useRef<HTMLSpanElement>(null)
  const isEmpty = !value

  const IconComponent = value ? iconMap[value] : undefined

  useEffect(() => {
    if (!showHelp) return

    const handleClickOutside = (e: MouseEvent) => {
      if (
        helpRef.current &&
        !helpRef.current.contains(e.target as Node) &&
        triggerRef.current &&
        !triggerRef.current.contains(e.target as Node)
      ) {
        setShowHelp(false)
      }
    }

    document.addEventListener("mousedown", handleClickOutside)
    return () => document.removeEventListener("mousedown", handleClickOutside)
  }, [showHelp])

  return (
    <div className="relative">
      <div className="flex items-center gap-2">
        <Input
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder="输入图标名称，如 Home、User"
          className="flex-1"
        />
        {IconComponent ? (
          <IconComponent className="h-5 w-5 text-primary" />
        ) : (
          <span
            ref={triggerRef}
            onClick={() => isEmpty && setShowHelp(!showHelp)}
            className="h-5 w-5 cursor-pointer text-center text-muted-foreground hover:text-foreground"
          >
            ?
          </span>
        )}
      </div>

      {showHelp && (
        <div
          ref={helpRef}
          className="absolute bottom-full left-0 z-50 mb-2 w-64 rounded-lg border bg-background p-3 shadow-lg"
        >
          <div className="space-y-1">
            <p className="font-medium">输入 Lucide 图标名称</p>
            <p className="text-muted-foreground">
              支持 1500+ 图标，如 Home、User、Settings
            </p>
            <div className="flex items-center gap-1">
              <Link href="https://lucide.dev/icons/" target="_blank">
                浏览所有图标
              </Link>
              <ExternalLinkIcon className="h-3 w-3 text-primary" />
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
