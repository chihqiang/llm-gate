import request, { PageRequest, PageResponse } from "@/lib/request"

// 日志信息
export interface Log {
  /** 日志ID */
  id: number
  /** 请求路径 */
  request_path: string
  /** 请求方法 */
  request_method: string
  /** 响应码 */
  response_code: number
  /** 请求参数 */
  request_payload?: string
  /** 请求IP */
  request_ip?: string
  /** 操作系统 */
  request_os?: string
  /** 浏览器 */
  request_browser?: string
  /** 响应数据 */
  response_json?: string
  /** 处理时间 */
  process_time?: string
  /** 账号ID */
  account_id?: number
  /** 账号名称 */
  account_name?: string
  /** 描述 */
  description?: string
}

// 日志列表请求
export interface LogListRequest extends PageRequest {
  id?: number
  request_ip?: string
  account_id?: number
  request_method?: string
  request_path?: string
}

// 日志列表响应
export interface LogListResponse extends PageResponse<Log> {}

// 获取日志列表
export async function logListApi(
  data: LogListRequest
): Promise<LogListResponse> {
  return await request.get<LogListResponse>("/api/v1/sys/logs", {
    params: data,
  })
}
