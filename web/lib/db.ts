import { ChatMessage } from "@/api/chat"

const DB_NAME = "llm-gate-chat"
const DB_VERSION = 1
const STORE_NAME = "conversations"

export interface Conversation {
  id: string
  title: string
  model: string
  messages: ChatMessage[]
  created_at: string
  updated_at: string
}

function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(DB_NAME, DB_VERSION)
    request.onupgradeneeded = () => {
      const db = request.result
      if (!db.objectStoreNames.contains(STORE_NAME)) {
        const store = db.createObjectStore(STORE_NAME, { keyPath: "id" })
        store.createIndex("updated_at", "updated_at", { unique: false })
      }
    }
    request.onsuccess = () => resolve(request.result)
    request.onerror = () => reject(request.error)
  })
}

export async function getAllConversations(): Promise<Conversation[]> {
  const db = await openDB()
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE_NAME, "readonly")
    const store = tx.objectStore(STORE_NAME)
    const index = store.index("updated_at")
    const request = index.getAll()
    request.onsuccess = () => {
      const list = request.result as Conversation[]
      list.sort((a, b) => b.updated_at.localeCompare(a.updated_at))
      resolve(list)
    }
    request.onerror = () => reject(request.error)
    tx.oncomplete = () => db.close()
  })
}

export async function getConversation(
  id: string
): Promise<Conversation | undefined> {
  const db = await openDB()
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE_NAME, "readonly")
    const store = tx.objectStore(STORE_NAME)
    const request = store.get(id)
    request.onsuccess = () => {
      resolve(request.result as Conversation | undefined)
    }
    request.onerror = () => reject(request.error)
    tx.oncomplete = () => db.close()
  })
}

export async function createConversation(
  data: Omit<Conversation, "id" | "created_at" | "updated_at">
): Promise<Conversation> {
  const now = new Date().toISOString()
  const conversation: Conversation = {
    id: crypto.randomUUID(),
    ...data,
    created_at: now,
    updated_at: now,
  }
  const db = await openDB()
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE_NAME, "readwrite")
    const store = tx.objectStore(STORE_NAME)
    const request = store.add(conversation)
    request.onsuccess = () => resolve(conversation)
    request.onerror = () => reject(request.error)
    tx.oncomplete = () => db.close()
  })
}

export async function updateConversation(
  id: string,
  data: Partial<Omit<Conversation, "id" | "created_at">>
): Promise<void> {
  const db = await openDB()
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE_NAME, "readwrite")
    const store = tx.objectStore(STORE_NAME)
    const getRequest = store.get(id)
    getRequest.onsuccess = () => {
      const existing = getRequest.result as Conversation | undefined
      if (!existing) {
        reject(new Error("Conversation not found"))
        return
      }
      const updated = { ...existing, ...data, updated_at: new Date().toISOString() }
      store.put(updated)
    }
    getRequest.onerror = () => reject(getRequest.error)
    tx.oncomplete = () => db.close()
    tx.onerror = () => reject(tx.error)
  })
}

export async function deleteConversation(id: string): Promise<void> {
  const db = await openDB()
  return new Promise((resolve, reject) => {
    const tx = db.transaction(STORE_NAME, "readwrite")
    const store = tx.objectStore(STORE_NAME)
    const request = store.delete(id)
    request.onsuccess = () => resolve()
    request.onerror = () => reject(request.error)
    tx.oncomplete = () => db.close()
  })
}
