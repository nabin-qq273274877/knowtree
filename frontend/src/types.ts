// 与后端 internal/api DTO 对应的类型定义（手工同步，API 面小可控）

export type NodeStatus =
  | 'not_started'
  | 'learning'
  | 'partial'
  | 'mastered'
  | 'partial_forgotten'
  | 'forgotten'

export type EdgeRelation = 'prerequisite' | 'related'

export interface KNode {
  id: string
  title: string
  content_md: string
  status: NodeStatus
  stage: string | null
  sort_order: number
  parent_id: string | null
  pos_x: number | null
  pos_y: number | null
  status_changed_at: number | null
  source_note: string | null
  created_at: number
  updated_at: number
  annotation_count: number
}

export interface KEdge {
  id: string
  source_id: string
  target_id: string
  relation: EdgeRelation
  label: string | null
  created_at: number
}

export interface KResource {
  id: string
  node_id: string
  kind: 'link' | 'file'
  title: string
  url: string | null
  path: string | null
  note: string | null
  created_at: number
}

export interface KAnnotation {
  id: string
  node_id: string
  content_md: string
  created_at: number
  updated_at: number
}

export interface VersionInfo {
  name: string
  version: string
  build_time: string
  commit: string
}
