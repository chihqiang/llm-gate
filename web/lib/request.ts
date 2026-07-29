import axios, {
  AxiosError,
  AxiosRequestConfig,
  AxiosResponse,
  InternalAxiosRequestConfig,
} from "axios"
import { getAccessToken, removeToken, emitUnauthorized } from "@/lib/token"

/**
 * 创建 Axios 实例
 * 统一配置基础信息：基础URL、超时时间、请求头格式
 */
const service = axios.create({
  // API 基础地址（从环境变量读取）
  baseURL: process.env.NEXT_PUBLIC_API_URL,
  // 请求超时时间：15秒
  timeout: 15000,
  // 默认请求头：提交数据为 JSON 格式
  headers: {
    "Content-Type": "application/json",
  },
})

// ==================== 全局类型定义 ====================
/**
 * 后端统一返回结构
 * @template T 业务数据类型
 */
export interface ApiResponse<T = unknown> {
  /** 业务状态码：0 = 成功，非0 = 失败 */
  code: number
  /** 返回提示信息 */
  msg: string
  /** 实际业务数据 */
  data: T
}

/**
 * 分页请求参数
 */
export interface PageRequest {
  page: number
  size: number
}

/**
 * 分页响应结构
 * @template T 列表项类型
 */
export interface PageResponse<T> {
  data: T[]
  total: number
}

// ==================== 请求拦截器 ====================
/**
 * 请求发送前统一处理
 * 1. 自动注入 Token
 */
service.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    // 从全局状态获取 Token
    const token = getAccessToken()

    // 如果 Token 存在，自动添加到请求头（Bearer 格式）
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`
    }

    return config
  },

  /**
   * 请求发送失败处理
   */
  (error: AxiosError) => {
    return Promise.reject(error)
  }
)

// ==================== 响应拦截器 ====================
/**
 * 响应成功后统一处理
 * 1. 处理业务错误（code !== 0）
 */
service.interceptors.response.use(
  (response: AxiosResponse<ApiResponse>) => {
    const res = response.data

    // 后端返回业务错误：抛出错误信息
    if (res.code !== 0) {
      return Promise.reject(new Error(res.msg || "请求失败"))
    }

    // 正常响应：返回数据
    return response
  },

  /**
   * 响应失败处理（HTTP 状态码错误：400/401/500 等）
   */
  (error: AxiosError) => {
    const status = error.response?.status
    const responseData = error.response?.data as ApiResponse | undefined

    // 统一错误信息：优先使用后端返回的 msg，其次使用 axios 错误信息
    const errMsg = responseData?.msg || error.message || "网络异常，请稍后重试"

    // 401 未授权 → 发出事件，由全局监听器处理
    if (status === 401) {
      removeToken()
      emitUnauthorized(errMsg)
    }

    // 控制台输出错误日志
    console.error("请求异常：", errMsg)

    // 抛出错误信息，让页面 catch 可以获取到
    return Promise.reject(new Error(errMsg))
  }
)

// ==================== 统一请求方法封装 ====================
/**
 * 文件上传配置
 */
export interface UploadConfig {
  /** 自定义请求头 */
  headers?: Record<string, string>
  /** 上传进度回调 */
  onProgress?: (progress: number) => void
  /** 超时时间（毫秒） */
  timeout?: number
}

/**
 * 单文件上传参数
 */
export interface UploadFileParams {
  /** 文件字段名 */
  fieldName?: string
  /** 文件对象 */
  file: File
  /** 额外表单字段 */
  fields?: Record<string, string | number | boolean>
}

/**
 * 多文件上传参数
 */
export interface UploadFilesParams {
  /** 文件字段名 */
  fieldName?: string
  /** 文件对象数组 */
  files: File[]
  /** 额外表单字段 */
  fields?: Record<string, string | number | boolean>
}

/**
 * 全局请求工具
 * 封装 GET/POST/PUT/DELETE/UPLOAD，自动解包 data.data
 * 业务页面直接获取数据，无需重复处理
 */
const request = {
  /**
   * GET 请求
   * @param url 请求地址
   * @param config 自定义配置
   * @returns 业务数据
   */
  get<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T> {
    return service.get<ApiResponse<T>>(url, config).then((res) => res.data.data)
  },

  /**
   * POST 请求
   * @param url 请求地址
   * @param data 请求体
   * @param config 自定义配置
   * @returns 业务数据
   */
  post<T = unknown>(
    url: string,
    data?: unknown,
    config?: AxiosRequestConfig
  ): Promise<T> {
    return service
      .post<ApiResponse<T>>(url, data, config)
      .then((res) => res.data.data)
  },

  /**
   * PUT 请求
   * @param url 请求地址
   * @param data 请求体
   * @param config 自定义配置
   * @returns 业务数据
   */
  put<T = unknown>(
    url: string,
    data?: unknown,
    config?: AxiosRequestConfig
  ): Promise<T> {
    return service
      .put<ApiResponse<T>>(url, data, config)
      .then((res) => res.data.data)
  },

  /**
   * DELETE 请求
   * @param url 请求地址
   * @param config 自定义配置
   * @returns 业务数据
   */
  delete<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T> {
    return service
      .delete<ApiResponse<T>>(url, config)
      .then((res) => res.data.data)
  },

  /**
   * 通用上传方法（使用 FormData）
   * @param url 请求地址
   * @param formData FormData 对象
   * @param config 上传配置
   * @returns 业务数据
   */
  upload<T = unknown>(
    url: string,
    formData: FormData,
    config?: UploadConfig
  ): Promise<T> {
    const axiosConfig: AxiosRequestConfig = {
      headers: {
        "Content-Type": "multipart/form-data",
        ...config?.headers,
      },
      timeout: config?.timeout || 60000,
    }

    if (config?.onProgress) {
      axiosConfig.onUploadProgress = (progressEvent) => {
        if (progressEvent.total) {
          const progress = Math.round(
            (progressEvent.loaded * 100) / progressEvent.total
          )
          config.onProgress!(progress)
        }
      }
    }

    return service
      .post<ApiResponse<T>>(url, formData, axiosConfig)
      .then((res) => res.data.data)
  },

  /**
   * 单文件上传
   * @param url 请求地址
   * @param params 上传参数（文件 + 额外字段）
   * @param config 上传配置
   * @returns 业务数据
   *
   * @example
   * ```tsx
   * const [file, setFile] = useState<File | null>(null)
   *
   * const handleUpload = async () => {
   *   if (!file) return
   *   const result = await request.uploadFile('/api/v1/upload', {
   *     file,
   *     fieldName: 'file',
   *     fields: { category: 'avatar', description: '用户头像' }
   *   }, {
   *     onProgress: (progress) => console.log(`上传进度: ${progress}%`)
   *   })
   * }
   * ```
   */
  uploadFile<T = unknown>(
    url: string,
    params: UploadFileParams,
    config?: UploadConfig
  ): Promise<T> {
    const formData = new FormData()
    const fieldName = params.fieldName || "file"

    formData.append(fieldName, params.file)

    if (params.fields) {
      for (const [key, value] of Object.entries(params.fields)) {
        formData.append(key, String(value))
      }
    }

    return this.upload<T>(url, formData, config)
  },

  /**
   * 多文件上传
   * @param url 请求地址
   * @param params 上传参数（文件数组 + 额外字段）
   * @param config 上传配置
   * @returns 业务数据
   *
   * @example
   * ```tsx
   * const [files, setFiles] = useState<File[]>([])
   *
   * const handleUpload = async () => {
   *   if (files.length === 0) return
   *   const result = await request.uploadFiles('/api/v1/upload/batch', {
   *     files,
   *     fieldName: 'files',
   *     fields: { category: 'images' }
   *   }, {
   *     onProgress: (progress) => console.log(`上传进度: ${progress}%`)
   *   })
   * }
   * ```
   */
  uploadFiles<T = unknown>(
    url: string,
    params: UploadFilesParams,
    config?: UploadConfig
  ): Promise<T> {
    const formData = new FormData()
    const fieldName = params.fieldName || "files"

    params.files.forEach((file) => {
      formData.append(fieldName, file)
    })

    if (params.fields) {
      for (const [key, value] of Object.entries(params.fields)) {
        formData.append(key, String(value))
      }
    }

    return this.upload<T>(url, formData, config)
  },
}

// 通用的多API调用方法
// 1. 同时调用多个 API，等待所有完成
// 2. 合并结果，返回一个对象，键为 API 名称，值为结果
// 3. 处理错误：捕获并抛出错误信息
export async function combineApiCalls<
  T extends Record<string, () => Promise<any>>,
>(calls: T): Promise<{ [K in keyof T]: Awaited<ReturnType<T[K]>> }> {
  const keys = Object.keys(calls) as Array<keyof T>
  const promises = keys.map((key) => calls[key]())
  const results = await Promise.all(promises)

  return keys.reduce(
    (acc, key, index) => {
      acc[key] = results[index]
      return acc
    },
    {} as { [K in keyof T]: Awaited<ReturnType<T[K]>> }
  )
}

// 串行调用多个 API，依次执行，等待每个完成后再执行下一个
// 用于需要依赖前置 API 结果的场景（如登录后获取用户信息）
export async function sequentialApiCalls<
  T extends Record<string, () => Promise<any>>,
>(calls: T): Promise<{ [K in keyof T]: Awaited<ReturnType<T[K]>> }> {
  const keys = Object.keys(calls) as Array<keyof T>
  const results = {} as { [K in keyof T]: Awaited<ReturnType<T[K]>> }

  for (const key of keys) {
    results[key] = await calls[key]()
  }

  return results
}

export default request
