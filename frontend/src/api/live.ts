export interface Activity {
  message: string
  file: string
  change?: string
  taskId?: string
  action: string
  timestamp: string
}

export interface LiveEvent {
  type: 'ready' | 'project_changed'
  projectRoot?: string
  activities?: Activity[]
  timestamp: string
}
