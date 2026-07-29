import { AuthAccount } from "@/lib/account"
import request, {
  sequentialApiCalls,
  PageRequest,
  PageResponse,
} from "@/lib/request"
import { AuthToken, setToken } from "@/lib/token"

export interface LoginRequest {
  email: string
  password: string
  remember: boolean
}

export interface AccountRole {
  id: number
  name: string
}

export interface Account {
  id: number
  name: string
  email: string
  roles: AccountRole[]
  status: boolean
}

export interface AccountCreateUpdate extends Account {
  password?: string
}

export interface AccountListRequest extends PageRequest {
  id?: number
}

export interface AccountListResponse extends PageResponse<Account> {}

export async function loginApi(data: LoginRequest): Promise<AuthToken> {
  return await request.post<AuthToken>("/api/v1/auth/login", data)
}

export async function userProfileApi(): Promise<AuthAccount> {
  return await request.get<AuthAccount>("/api/v1/auth/me")
}

export async function accountListApi(
  data: AccountListRequest
): Promise<AccountListResponse> {
  return await request.get<AccountListResponse>("/api/v1/sys/accounts", {
    params: data,
  })
}

export async function accountDetailApi(id: number): Promise<Account> {
  return await request.get<Account>(`/api/v1/sys/accounts/${id}`)
}

export async function accountCreateApi(
  data: AccountCreateUpdate
): Promise<Account> {
  return await request.post<Account>("/api/v1/sys/accounts", data)
}

export async function accountUpdateApi(
  data: AccountCreateUpdate
): Promise<Account> {
  return await request.put<Account>(`/api/v1/sys/accounts/${data.id}`, data)
}

export async function accountDeleteApi(id: number): Promise<Account> {
  return await request.delete<Account>(`/api/v1/sys/accounts/${id}`)
}
