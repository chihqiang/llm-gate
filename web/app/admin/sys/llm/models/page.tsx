"use client"

import {
  ModelConfig,
  modelListApi,
  modelCreateApi,
  modelUpdateApi,
  modelDeleteApi,
  modelDetailApi,
} from "@/api/llm"
import { ModelForm } from "@/components/forms/model-form"
import { Crud, SearchField } from "@/components/widgets/crud"
import { DataListColumn } from "@/components/widgets/data-list"

const searchFields: SearchField[] = [
  {
    name: "name",
    label: "模型名称",
    type: "input",
    placeholder: "搜索模型名称",
  },
]

const columns: DataListColumn<ModelConfig>[] = [
  {
    key: "id",
    header: "ID",
    cellClassName: "w-16 font-medium",
    cell: (row) => row.id,
  },
  {
    key: "name",
    header: "模型名称",
    cell: (row) => row.name,
  },
  {
    key: "provider_id",
    header: "服务商ID",
    cell: (row) => row.provider_id,
  },
  {
    key: "upstream_model_name",
    header: "上游模型",
    cell: (row) => row.upstream_model_name,
  },
  {
    key: "model_ratio",
    header: "倍率",
    cell: (row) => row.model_ratio,
  },
  {
    key: "status",
    header: "状态",
    cell: (row) => (row.status ? "启用" : "禁用"),
  },
]

const defaultFormData: ModelConfig = {
  id: 0,
  name: "",
  provider_id: 0,
  upstream_model_name: "",
  model_ratio: 1,
  completion_ratio: 1,
  status: true,
  remark: "",
  created_at: "",
  updated_at: "",
}

export default function ModelsPage() {
  return (
    <Crud<ModelConfig, ModelConfig>
      title="模型管理"
      entityName="模型"
      columns={columns}
      listApi={modelListApi}
      pageSize={8}
      searchFields={searchFields}
      deleteApi={modelDeleteApi}
      formComponent={ModelForm}
      defaultFormData={defaultFormData}
      createApi={modelCreateApi}
      updateApi={modelUpdateApi}
      detailApi={modelDetailApi}
    />
  )
}
