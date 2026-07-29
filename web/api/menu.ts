import request, { PageRequest, PageResponse } from "@/lib/request"

export const menuTypeMap: Record<number, string> = {
  1: "目录",
  2: "菜单",
  3: "按钮",
}

export const apiMethodMap: Record<string, string> = {
  "": "无",
  GET: "GET",
  POST: "POST",
  PUT: "PUT",
  DELETE: "DELETE",
  "*": "不限",
}

export interface Menu {
  id: number
  pid: number
  menu_type: number
  name: string
  path: string
  component: string
  icon: string
  sort: number
  api_url: string
  api_method: string
  visible: boolean
  status: boolean
  remark: string
}

export interface MenuListRequest extends PageRequest {
  id?: number
}

export interface MenuListResponse extends PageResponse<Menu> {}

export async function menuListApi(
  data: MenuListRequest
): Promise<MenuListResponse> {
  return await request.get<MenuListResponse>("/api/v1/sys/menus", {
    params: data,
  })
}

export async function menuDetailApi(id: number): Promise<Menu> {
  return await request.get<Menu>(`/api/v1/sys/menus/${id}`)
}

export async function menuCreateApi(data: Menu): Promise<Menu> {
  return await request.post<Menu>("/api/v1/sys/menus", data)
}

export async function menuUpdateApi(data: Menu): Promise<Menu> {
  return await request.put<Menu>(`/api/v1/sys/menus/${data.id}`, data)
}

export async function menuDeleteApi(id: number): Promise<Menu> {
  return await request.delete<Menu>(`/api/v1/sys/menus/${id}`)
}

export async function menuAllApi(): Promise<Menu[]> {
  return await request.get<Menu[]>("/api/v1/sys/menus/all")
}
