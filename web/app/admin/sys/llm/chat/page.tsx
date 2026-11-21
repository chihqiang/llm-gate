"use client"

import { useState, useRef, useEffect, useCallback } from "react"
import ReactMarkdown from "react-markdown"
import remarkGfm from "remark-gfm"
import rehypeHighlight from "rehype-highlight"
import {
  ArrowUp,
  Bot,
  User,
  Loader2,
  Trash2,
  MessageSquare,
  Plus,
  ChevronLeft,
  ChevronRight,
  Pencil,
  Check,
  X,
  Copy,
  CheckCheck,
  RotateCcw,
  Trash,
  Square,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { cn } from "@/lib/utils"
import {
  ChatMessage,
  ChatCompletionUsage,
  RelayModel,
  listRelayModelsApi,
  chatCompletionStream,
} from "@/api/chat"
import { tokenListApi, tokenRevealApi } from "@/api/llm"
import type { UserToken } from "@/api/llm"
import {
  Conversation,
  getAllConversations,
  getConversation,
  createConversation,
  updateConversation,
  deleteConversation,
} from "@/lib/db"

const WELCOME_MSG: ChatMessage = {
  role: "assistant",
  content: "你好！有什么可以帮助你的？",
}

function generateTitle(messages: ChatMessage[]): string {
  const firstUser = messages.find((m) => m.role === "user")
  if (!firstUser) return "新对话"
  const t = firstUser.content.trim()
  return t.length > 30 ? t.slice(0, 30) + "..." : t
}

function formatTokens(n: number): string {
  if (n >= 1000) return (n / 1000).toFixed(1) + "k"
  return n.toString()
}

function CodeBlock({
  className,
  children,
}: {
  className?: string
  children?: React.ReactNode
}) {
  const [copied, setCopied] = useState(false)
  const code = typeof children === "string" ? children : ""
  const lang = className?.replace("language-", "") || ""

  const handleCopy = async () => {
    await navigator.clipboard.writeText(code)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="group relative my-3 overflow-hidden rounded-lg border bg-muted/50">
      <div className="flex items-center justify-between border-b bg-muted/30 px-4 py-1.5 text-xs text-muted-foreground">
        <span>{lang || "code"}</span>
        <button
          onClick={handleCopy}
          className="flex items-center gap-1 text-muted-foreground transition-colors hover:text-foreground"
        >
          {copied ? (
            <CheckCheck className="h-3.5 w-3.5" />
          ) : (
            <Copy className="h-3.5 w-3.5" />
          )}
          {copied ? "已复制" : "复制"}
        </button>
      </div>
      <pre className="overflow-x-auto p-4 text-sm leading-relaxed">
        {children}
      </pre>
    </div>
  )
}

function MessageContent({ content }: { content: string }) {
  return (
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      rehypePlugins={[rehypeHighlight]}
      components={{
        code({ className, children, ...props }) {
          const match = /language-(\w+)/.exec(className || "")
          const isInline = !match && !className
          if (isInline) {
            return (
              <code
                className="rounded bg-muted/80 px-1.5 py-0.5 font-mono text-sm"
                {...props}
              >
                {children}
              </code>
            )
          }
          return (
            <CodeBlock className={className}>
              {String(children).replace(/\n$/, "")}
            </CodeBlock>
          )
        },
        pre({ children }) {
          return <>{children}</>
        },
        a({ href, children }) {
          return (
            <a
              href={href}
              target="_blank"
              rel="noreferrer"
              className="text-primary underline underline-offset-2 hover:no-underline"
            >
              {children}
            </a>
          )
        },
      }}
    >
      {content}
    </ReactMarkdown>
  )
}

function UsageBadge({ usage }: { usage: ChatCompletionUsage }) {
  return (
    <div className="mt-2 flex items-center gap-3 text-xs text-muted-foreground">
      <span>↑ {formatTokens(usage.prompt_tokens)}</span>
      <span>↓ {formatTokens(usage.completion_tokens)}</span>
      <span className="font-medium">∑ {formatTokens(usage.total_tokens)}</span>
    </div>
  )
}

export default function ChatPage() {
  const [models, setModels] = useState<RelayModel[]>([])
  const [selectedModel, setSelectedModel] = useState("")
  const [messages, setMessages] = useState<ChatMessage[]>([WELCOME_MSG])
  const [input, setInput] = useState("")
  const [streaming, setStreaming] = useState(false)
  const [loadingModels, setLoadingModels] = useState(true)
  const [lastUsage, setLastUsage] = useState<ChatCompletionUsage | undefined>()

  const [conversations, setConversations] = useState<Conversation[]>([])
  const [activeId, setActiveId] = useState<string | null>(null)
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editingTitle, setEditingTitle] = useState("")
  const [loadingConvs, setLoadingConvs] = useState(true)

  const [tokens, setTokens] = useState<UserToken[]>([])
  const [selectedTokenId, setSelectedTokenId] = useState<number | null>(null)
  const [revealedKey, setRevealedKey] = useState("")
  const [loadingTokens, setLoadingTokens] = useState(true)
  const [revealingToken, setRevealingToken] = useState(false)

  const messagesEndRef = useRef<HTMLDivElement>(null)
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const abortRef = useRef<AbortController | null>(null)
  const messagesContainerRef = useRef<HTMLDivElement>(null)
  const activeIdRef = useRef(activeId)
  const messagesRef = useRef(messages)
  const streamingRef = useRef(streaming)
  const autoScrollRef = useRef(true)

  activeIdRef.current = activeId
  messagesRef.current = messages
  streamingRef.current = streaming

  const scrollToBottom = useCallback(() => {
    if (!autoScrollRef.current) return
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" })
  }, [])

  useEffect(() => {
    scrollToBottom()
  }, [messages, scrollToBottom])

  useEffect(() => {
    const el = messagesContainerRef.current
    if (!el) return
    const handleScroll = () => {
      const threshold = 100
      autoScrollRef.current =
        el.scrollHeight - el.scrollTop - el.clientHeight < threshold
    }
    el.addEventListener("scroll", handleScroll)
    return () => el.removeEventListener("scroll", handleScroll)
  }, [])

  useEffect(() => {
    listRelayModelsApi(revealedKey || "none")
      .then((list) => {
        setModels(list)
        if (list.length > 0) setSelectedModel(list[0].id)
      })
      .catch(() => setModels([]))
      .finally(() => setLoadingModels(false))
  }, [revealedKey])

  useEffect(() => {
    getAllConversations()
      .then(setConversations)
      .finally(() => setLoadingConvs(false))
  }, [])

  useEffect(() => {
    tokenListApi({ page: 1, size: 100 })
      .then((res) => {
        const active = res.data.filter((t) => t.status)
        setTokens(active)
      })
      .catch(() => setTokens([]))
      .finally(() => setLoadingTokens(false))
  }, [])

  useEffect(() => {
    if (textareaRef.current) {
      textareaRef.current.style.height = "auto"
      textareaRef.current.style.height =
        Math.min(textareaRef.current.scrollHeight, 200) + "px"
    }
  }, [input])

  async function switchConversation(id: string | null) {
    if (streamingRef.current) {
      abortRef.current?.abort()
      setStreaming(false)
      abortRef.current = null
    }
    if (activeIdRef.current) {
      const msgs = messagesRef.current
      if (msgs.some((m) => m.role === "user")) {
        await updateConversation(activeIdRef.current, {
          messages: msgs,
          model: selectedModel,
          title: generateTitle(msgs),
        })
        const updated = await getAllConversations()
        setConversations(updated)
      }
    }
    if (!id) {
      setActiveId(null)
      setMessages([WELCOME_MSG])
      setLastUsage(undefined)
      return
    }
    const conv = await getConversation(id)
    if (!conv) return
    setActiveId(id)
    setSelectedModel(conv.model || models[0]?.id || "")
    setMessages(conv.messages.length > 0 ? conv.messages : [WELCOME_MSG])
    setLastUsage(undefined)
  }

  async function handleNew() {
    await switchConversation(null)
  }

  async function handleSelect(id: string) {
    await switchConversation(id)
  }

  async function handleDelete(id: string, e: React.MouseEvent) {
    e.stopPropagation()
    if (streamingRef.current) {
      abortRef.current?.abort()
      setStreaming(false)
      abortRef.current = null
    }
    await deleteConversation(id)
    setConversations((prev) => prev.filter((c) => c.id !== id))
    if (activeIdRef.current === id) {
      setActiveId(null)
      setMessages([WELCOME_MSG])
    }
  }

  function startEditTitle(conv: Conversation, e: React.MouseEvent) {
    e.stopPropagation()
    setEditingId(conv.id)
    setEditingTitle(conv.title)
  }

  async function confirmEditTitle() {
    if (editingId && editingTitle.trim()) {
      await updateConversation(editingId, { title: editingTitle.trim() })
      setConversations((prev) =>
        prev.map((c) =>
          c.id === editingId ? { ...c, title: editingTitle.trim() } : c
        )
      )
    }
    setEditingId(null)
    setEditingTitle("")
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  async function sendMessage(msgs: ChatMessage[]) {
    if (!revealedKey) {
      setMessages((prev) => {
        const updated = [...prev]
        const last = updated[updated.length - 1]
        if (last.role === "assistant" && !last.content) {
          updated[updated.length - 1] = {
            ...last,
            content: "请先选择一个 API Key",
          }
        }
        return updated
      })
      setStreaming(false)
      return
    }

    setStreaming(true)
    setLastUsage(undefined)

    const sendMessages = msgs.filter((m) => m.role !== "system" || m.content)

    try {
      abortRef.current = await chatCompletionStream(
        { model: selectedModel, messages: sendMessages },
        revealedKey,
        (chunk) => {
          const delta = chunk.choices?.[0]?.delta?.content
          if (delta) {
            setMessages((prev) => {
              const updated = [...prev]
              const last = updated[updated.length - 1]
              if (last.role === "assistant") {
                updated[updated.length - 1] = {
                  ...last,
                  content: last.content + delta,
                }
              }
              return updated
            })
          }
        },
        (usage) => {
          setStreaming(false)
          abortRef.current = null
          if (usage) setLastUsage(usage)
          persistConversation()
        },
        (err) => {
          setMessages((prev) => {
            const updated = [...prev]
            const last = updated[updated.length - 1]
            if (last.role === "assistant") {
              updated[updated.length - 1] = {
                ...last,
                content: last.content || "请求失败: " + err.message,
              }
            }
            return updated
          })
          setStreaming(false)
          abortRef.current = null
          persistConversation()
        }
      )
    } catch {
      setStreaming(false)
    }
  }

  async function persistConversation() {
    const id = activeIdRef.current
    if (!id) return
    const msgs = messagesRef.current
    if (!msgs.some((m) => m.role === "user")) return
    await updateConversation(id, {
      messages: msgs,
      model: selectedModel,
      title: generateTitle(msgs),
    })
    setConversations((prev) => {
      const updated = prev.map((c) =>
        c.id === id
          ? {
              ...c,
              messages: msgs,
              model: selectedModel,
              title: generateTitle(msgs),
              updated_at: new Date().toISOString(),
            }
          : c
      )
      updated.sort((a, b) => b.updated_at.localeCompare(a.updated_at))
      return updated
    })
  }

  async function handleSend() {
    const content = input.trim()
    if (!content || !selectedModel || streaming) return
    setInput("")

    let convId = activeId
    const hasHistory = messages.some((m) => m.role === "user")

    if (!convId) {
      const conv = await createConversation({
        title: generateTitle([...messages, { role: "user", content }]),
        model: selectedModel,
        messages: [],
      })
      convId = conv.id
      setActiveId(convId)
      setConversations((prev) => [
        {
          ...conv,
          title: generateTitle([...messages, { role: "user", content }]),
        },
        ...prev,
      ])
    } else if (!hasHistory) {
      await updateConversation(convId, {
        title: generateTitle([...messages, { role: "user", content }]),
        model: selectedModel,
      })
    }

    const userMsg: ChatMessage = { role: "user", content }
    const assistantMsg: ChatMessage = { role: "assistant", content: "" }
    const updatedMessages = [...messages, userMsg, assistantMsg]
    setMessages(updatedMessages)

    await sendMessage(updatedMessages)
  }

  async function handleRegenerate() {
    if (streaming) return
    const idx = messages.length - 1
    if (idx < 0 || messages[idx].role !== "assistant") return

    const prevMessages = messages.slice(0, -1)
    const emptyMsg: ChatMessage = { role: "assistant", content: "" }
    setMessages([...prevMessages, emptyMsg])
    setLastUsage(undefined)

    await sendMessage([...prevMessages, emptyMsg])
  }

  function handleStop() {
    abortRef.current?.abort()
    setStreaming(false)
    abortRef.current = null
  }

  async function handleDeleteMessage(idx: number) {
    if (streaming) return
    const updated = messages.filter((_, i) => i !== idx)
    setMessages(updated.length === 0 ? [WELCOME_MSG] : updated)
    if (activeIdRef.current) {
      const final = updated.length === 0 ? [WELCOME_MSG] : updated
      await updateConversation(activeIdRef.current, {
        messages: final,
        title: generateTitle(final),
      })
    }
  }

  async function handleCopyMessage(content: string) {
    await navigator.clipboard.writeText(content)
  }

  async function handleKeySelect(v: string) {
    if (!v) return
    const id = Number(v)
    setSelectedTokenId(id)
    setRevealedKey("")
    setRevealingToken(true)
    try {
      const key = await tokenRevealApi(id)
      setRevealedKey(key || "")
    } catch {
      // ignore
    } finally {
      setRevealingToken(false)
    }
  }

  return (
    <div className="flex h-[calc(100vh-12rem)] gap-0 overflow-hidden">
      {/* Sidebar */}
      <div
        className={cn(
          "flex flex-col overflow-hidden border-r transition-all duration-200",
          sidebarOpen ? "w-60 shrink-0" : "w-0 border-r-0"
        )}
      >
        <div
          className={cn(
            "flex h-full flex-col",
            sidebarOpen ? "min-w-60" : "hidden"
          )}
        >
          <div className="border-b p-3">
            <Button
              variant="outline"
              className="w-full justify-start gap-2"
              onClick={handleNew}
            >
              <Plus className="h-4 w-4" /> 新对话
            </Button>
          </div>
          <div className="flex-1 space-y-1 overflow-y-auto p-2">
            {loadingConvs ? (
              <div className="space-y-2 p-2">
                {[1, 2, 3].map((i) => (
                  <div
                    key={i}
                    className="h-8 animate-pulse rounded-md bg-muted"
                  />
                ))}
              </div>
            ) : conversations.length === 0 ? (
              <div className="flex flex-col items-center justify-center py-10 text-muted-foreground">
                <MessageSquare className="mb-2 h-8 w-8 opacity-30" />
                <p className="text-sm">暂无历史对话</p>
              </div>
            ) : (
              conversations.map((conv) => (
                <div
                  key={conv.id}
                  onClick={() => handleSelect(conv.id)}
                  className={cn(
                    "group flex cursor-pointer items-center gap-2 rounded-md border-l-2 px-3 py-1.5 text-sm transition-colors",
                    activeId === conv.id
                      ? "border-l-primary bg-primary/5 text-primary"
                      : "border-l-transparent hover:bg-muted"
                  )}
                >
                  <MessageSquare className="h-4 w-4 shrink-0" />
                  {editingId === conv.id ? (
                    <div className="flex flex-1 items-center gap-1">
                      <Input
                        value={editingTitle}
                        onChange={(e) => setEditingTitle(e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === "Enter") confirmEditTitle()
                          if (e.key === "Escape") setEditingId(null)
                        }}
                        className="h-7 px-1 py-0 text-sm"
                        onClick={(e) => e.stopPropagation()}
                        autoFocus
                      />
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-6 w-6 shrink-0"
                        onClick={(e) => {
                          e.stopPropagation()
                          confirmEditTitle()
                        }}
                      >
                        <Check className="h-3 w-3" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-6 w-6 shrink-0"
                        onClick={(e) => {
                          e.stopPropagation()
                          setEditingId(null)
                        }}
                      >
                        <X className="h-3 w-3" />
                      </Button>
                    </div>
                  ) : (
                    <span className="flex-1 truncate">{conv.title}</span>
                  )}
                  {editingId !== conv.id && (
                    <div className="hidden items-center gap-0.5 group-hover:flex">
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-6 w-6"
                        onClick={(e) => startEditTitle(conv, e)}
                      >
                        <Pencil className="h-3 w-3" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-6 w-6 text-destructive hover:text-destructive"
                        onClick={(e) => handleDelete(conv.id, e)}
                      >
                        <Trash2 className="h-3 w-3" />
                      </Button>
                    </div>
                  )}
                </div>
              ))
            )}
          </div>
        </div>
      </div>

      {/* Toggle sidebar */}
      <Button
        variant="ghost"
        size="icon"
        className="z-10 mt-2 -ml-4 h-8 w-8 shrink-0 rounded-full border bg-background shadow-sm"
        onClick={() => setSidebarOpen(!sidebarOpen)}
      >
        {sidebarOpen ? (
          <ChevronLeft className="h-4 w-4" />
        ) : (
          <ChevronRight className="h-4 w-4" />
        )}
      </Button>

      {/* Main chat area */}
      <div className="flex min-w-0 flex-1 flex-col bg-muted/5 px-4">
        {/* Header */}
        <div className="mb-4 flex items-center gap-2 pb-3">
          <Bot className="h-5 w-5 shrink-0 text-primary" />
          <h2 className="text-base font-semibold">在线聊天</h2>
        </div>

        {/* Messages */}
        <div
          ref={messagesContainerRef}
          className="mb-4 flex-1 space-y-4 overflow-y-auto pr-2"
        >
          {messages.map((msg, i) => (
            <div
              key={i}
              className={cn(
                "group flex gap-3",
                msg.role === "user" ? "justify-end" : "justify-start"
              )}
            >
              {msg.role === "assistant" && (
                <div className="mt-1 flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary/10">
                  <Bot className="h-4 w-4 text-primary" />
                </div>
              )}
              <div
                className={cn(
                  "max-w-[75%] space-y-1",
                  msg.role === "user" && "flex flex-col items-end"
                )}
              >
                <div
                  className={cn(
                    "rounded-lg px-4 py-2.5 text-sm leading-relaxed",
                    msg.role === "user"
                      ? "bg-primary text-primary-foreground shadow-sm"
                      : "border bg-muted shadow-sm"
                  )}
                >
                  {msg.role === "assistant" ? (
                    msg.content ? (
                      <MessageContent content={msg.content} />
                    ) : streaming && i === messages.length - 1 ? (
                      <span className="inline-flex gap-1 py-1">
                        <span className="h-2 w-2 animate-bounce rounded-full bg-muted-foreground/50 [animation-delay:0ms]" />
                        <span className="h-2 w-2 animate-bounce rounded-full bg-muted-foreground/50 [animation-delay:150ms]" />
                        <span className="h-2 w-2 animate-bounce rounded-full bg-muted-foreground/50 [animation-delay:300ms]" />
                      </span>
                    ) : null
                  ) : (
                    <span className="whitespace-pre-wrap">{msg.content}</span>
                  )}
                </div>
                {msg.role === "assistant" &&
                  !streaming &&
                  i === messages.length - 1 &&
                  lastUsage && <UsageBadge usage={lastUsage} />}
                {/* Message actions */}
                {msg.content && !streaming && (
                  <div
                    className={cn(
                      "flex gap-1 opacity-0 transition-opacity group-hover:opacity-100",
                      msg.role === "user" ? "justify-end" : "justify-start"
                    )}
                  >
                    {msg.content && (
                      <button
                        onClick={() => handleCopyMessage(msg.content)}
                        className="flex items-center gap-1 rounded px-1 py-0.5 text-xs text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                      >
                        <Copy className="h-3 w-3" /> 复制
                      </button>
                    )}
                    {msg.role === "assistant" && i === messages.length - 1 && (
                      <button
                        onClick={handleRegenerate}
                        className="flex items-center gap-1 rounded px-1 py-0.5 text-xs text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                        disabled={streaming}
                      >
                        <RotateCcw className="h-3 w-3" /> 重新生成
                      </button>
                    )}
                    {messages.length > 1 && (
                      <button
                        onClick={() => handleDeleteMessage(i)}
                        className="flex items-center gap-1 rounded px-1 py-0.5 text-xs text-destructive/70 transition-colors hover:bg-destructive/10 hover:text-destructive"
                      >
                        <Trash className="h-3 w-3" /> 删除
                      </button>
                    )}
                  </div>
                )}
              </div>
              {msg.role === "user" && (
                <div className="mt-1 flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary">
                  <User className="h-4 w-4 text-primary-foreground" />
                </div>
              )}
            </div>
          ))}
          <div ref={messagesEndRef} />
        </div>

        {/* Toolbar + Input */}
        <div className="border-t pt-3 pb-2">
          <div className="mb-3 flex items-center gap-2 px-0.5">
            <Select
              value={selectedTokenId?.toString() || ""}
              onValueChange={(v) => v && handleKeySelect(v)}
              disabled={loadingTokens || streaming || revealingToken}
            >
              <SelectTrigger className="h-7 w-auto gap-1 text-xs">
                <SelectValue
                  placeholder={loadingTokens ? "加载中..." : "API Key"}
                />
              </SelectTrigger>
              <SelectContent>
                {tokens.map((t) => (
                  <SelectItem key={t.id} value={t.id.toString()}>
                    <span className="text-xs">
                      {t.name} ({t.key_masked})
                    </span>
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {revealingToken && (
              <Loader2 className="h-3 w-3 animate-spin text-muted-foreground" />
            )}
            <Select
              value={selectedModel}
              onValueChange={(v) => v && setSelectedModel(v)}
              disabled={loadingModels || streaming || !revealedKey}
            >
              <SelectTrigger className="h-7 w-auto gap-1 text-xs">
                <SelectValue
                  placeholder={
                    loadingModels
                      ? "加载中..."
                      : !revealedKey
                        ? "请选 API Key"
                        : "选择模型"
                  }
                />
              </SelectTrigger>
              <SelectContent>
                {models.length === 0 && !loadingModels && (
                  <div className="px-2 py-4 text-center text-xs text-muted-foreground">
                    暂无可用模型
                  </div>
                )}
                {models.map((m) => (
                  <SelectItem key={m.id} value={m.id}>
                    {m.id}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="relative flex items-end rounded-xl border bg-card px-4 py-3 shadow-sm transition-all focus-within:border-ring/50 focus-within:shadow-md">
            <Textarea
              ref={textareaRef}
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder={
                !revealedKey
                  ? "请先选择 API Key"
                  : "输入消息，Enter 发送，Shift+Enter 换行"
              }
              disabled={streaming || !selectedModel || !revealedKey}
              rows={1}
              className="max-h-48 min-h-[44px] resize-none border-0 bg-transparent p-0 pr-12 focus-visible:ring-0 disabled:opacity-60"
            />
            {streaming ? (
              <Button
                variant="destructive"
                size="icon"
                onClick={handleStop}
                title="停止生成"
                className="absolute right-2 bottom-2 h-8 w-8 rounded-full"
              >
                <Square className="h-4 w-4 fill-current" />
              </Button>
            ) : (
              <Button
                size="icon"
                onClick={handleSend}
                disabled={!input.trim() || !selectedModel || !revealedKey}
                title="发送"
                className="absolute right-2 bottom-2 h-8 w-8 rounded-full disabled:opacity-40"
              >
                <ArrowUp className="h-4 w-4" />
              </Button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
