"use client"

import request from "@/lib/request"

export interface RelayModel {
  id: string
  object: string
  created: number
  owned_by: string
}

export interface RelayModelList {
  object: string
  data: RelayModel[]
}

export interface ChatMessage {
  role: "user" | "assistant" | "system"
  content: string
}

export interface ChatCompletionRequest {
  model: string
  messages: ChatMessage[]
  stream?: boolean
  temperature?: number
  max_tokens?: number
}

export interface ChatCompletionUsage {
  prompt_tokens: number
  completion_tokens: number
  total_tokens: number
}

export interface StreamChunk {
  id: string
  object: string
  created: number
  model: string
  choices: {
    index: number
    delta: { content?: string; role?: string }
    finish_reason: string | null
  }[]
  usage?: ChatCompletionUsage
}

export async function listRelayModelsApi(apiKey: string): Promise<RelayModel[]> {
  const baseUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"
  const res = await fetch(`${baseUrl}/v1/models`, {
    headers: { Authorization: `Bearer ${apiKey}` },
  })
  if (!res.ok) throw new Error("Failed to fetch models")
  const body = await res.json()
  return body.data?.data || []
}

export async function chatCompletionStream(
  data: ChatCompletionRequest,
  apiKey: string,
  onChunk: (chunk: StreamChunk) => void,
  onDone: (usage?: ChatCompletionUsage) => void,
  onError: (err: Error) => void
): Promise<AbortController> {
  const controller = new AbortController()
  const baseUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"

  fetch(`${baseUrl}/v1/chat/completions`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${apiKey}`,
    },
    body: JSON.stringify({ ...data, stream: true }),
    signal: controller.signal,
  })
    .then(async (response) => {
      if (!response.ok) {
        const errBody = await response.text()
        throw new Error(errBody || `HTTP ${response.status}`)
      }
      const reader = response.body?.getReader()
      if (!reader) {
        throw new Error("Response body is not readable")
      }

      const decoder = new TextDecoder()
      let buffer = ""
      let lastUsage: ChatCompletionUsage | undefined

      function processLines() {
        const lines = buffer.split("\n")
        buffer = lines.pop() || ""
        for (const line of lines) {
          const trimmed = line.trim()
          if (!trimmed || trimmed === "data: [DONE]") {
            continue
          }
          if (trimmed.startsWith("data: ")) {
            try {
              const chunk = JSON.parse(trimmed.slice(6)) as StreamChunk
              if (chunk.usage) {
                lastUsage = chunk.usage
              }
              if (chunk.choices?.[0]?.delta?.content || chunk.choices?.[0]?.finish_reason) {
                onChunk(chunk)
              }
            } catch {
              // skip malformed chunk
            }
          }
        }
      }

      function pump(): Promise<void> {
        return reader!.read().then(({ done, value }) => {
          if (done) {
            processLines()
            onDone(lastUsage)
            return
          }
          buffer += decoder.decode(value, { stream: true })
          processLines()
          return pump()
        })
      }

      return pump()
    })
    .catch((err) => {
      if (err.name !== "AbortError") {
        onError(err instanceof Error ? err : new Error(String(err)))
      }
    })

  return controller
}
