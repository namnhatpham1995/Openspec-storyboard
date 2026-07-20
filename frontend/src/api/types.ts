export type ChangeStatus = 'draft' | 'in_progress' | 'complete' | 'archived'

export interface Artifacts {
  proposal: boolean
  specs: boolean
  design: boolean
  tasks: boolean
}

export interface Task {
  id: string
  text: string
  checked: boolean
  line: number
}

export interface TaskGroup {
  heading: string
  tasks: Task[] | null
}

export interface TasksDoc {
  groups: TaskGroup[] | null
  parseable: boolean
}

export interface Change {
  name: string
  archived: boolean
  artifacts: Artifacts
  tasks: TasksDoc
  status: ChangeStatus
}

export interface Project {
  root: string
  schemaName: string
  changes: Change[] | null
}

export interface FileVersion {
  modTime: string
  hash: string
}

export interface ArtifactFile {
  kind: string
  path: string
  content: string
  version: FileVersion
}

export interface ChangeDetail extends Change {
  artifactFiles: ArtifactFile[]
}

export interface ToggleResult {
  task: Task
  version: FileVersion
}

export interface TaskTextResult {
  task: Task
  version: FileVersion
}

export interface ArtifactWriteResult {
  artifact: ArtifactFile
}
