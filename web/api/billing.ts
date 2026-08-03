import request, { PageRequest, PageResponse } from "@/lib/request"

// ==================== 充值订单 ====================

export type RechargeOrderStatus = "pending" | "paid" | "cancelled"

export interface RechargeOrder {
  id: number
  account_id: number
  amount_cents: number
  status: RechargeOrderStatus
  remark: string
  created_by: number
  paid_by: number
  paid_at: string | null
  created_at: string
  updated_at: string
}

export interface OrderListRequest extends PageRequest {
  account_id?: number
  status?: RechargeOrderStatus
}

export interface OrderListResponse extends PageResponse<RechargeOrder> {}

export async function orderListApi(
  data: OrderListRequest
): Promise<OrderListResponse> {
  return await request.get<OrderListResponse>("/api/v1/billing/orders", {
    params: data,
  })
}

export interface OrderCreateRequest {
  account_id: number
  amount_cents: number
  remark?: string
}

export async function orderCreateApi(
  data: OrderCreateRequest
): Promise<RechargeOrder> {
  return await request.post<RechargeOrder>("/api/v1/billing/orders", data)
}

export async function orderConfirmApi(id: number): Promise<void> {
  return await request.post(`/api/v1/billing/orders/${id}/confirm`)
}

export async function orderCancelApi(id: number): Promise<void> {
  return await request.post(`/api/v1/billing/orders/${id}/cancel`)
}

// ==================== 资金流水 ====================

export type TransactionType = "consume" | "refund" | "recharge" | "adjust"

export interface BalanceTransaction {
  id: number
  account_id: number
  type: TransactionType
  amount_cents: number
  balance_cents: number
  token_id: number
  request_id: string
  remark: string
  created_at: string
}

export interface TransactionListRequest extends PageRequest {
  account_id?: number
  type?: TransactionType
}

export interface TransactionListResponse
  extends PageResponse<BalanceTransaction> {}

export async function transactionListApi(
  data: TransactionListRequest
): Promise<TransactionListResponse> {
  return await request.get<TransactionListResponse>(
    "/api/v1/billing/transactions",
    { params: data }
  )
}

// ==================== 余额调整 ====================

export interface BalanceAdjustRequest {
  account_id: number
  amount_cents: number
  remark?: string
}

export async function balanceAdjustApi(
  data: BalanceAdjustRequest
): Promise<void> {
  return await request.post("/api/v1/billing/balance/adjust", data)
}
