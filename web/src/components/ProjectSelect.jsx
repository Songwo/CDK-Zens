import { EmptyState } from "./ui";

export default function ProjectSelect({
  projects = [],
  value,
  onChange,
  locked = false,
  loading = false,
  onCreateProject,
  label = "所属项目",
}) {
  if (loading) {
    return (
      <div className="field-group">
        <label>{label} <b>*</b></label>
        <div className="empty-panel empty-panel--compact">正在加载项目...</div>
      </div>
    );
  }

  if (!projects.length) {
    return (
      <div className="field-group">
        <label>{label} <b>*</b></label>
        <EmptyState
          title="暂无项目，请先创建项目"
          description="活动必须归属于项目，用来统一管理库存、节点和领取记录。"
          action={onCreateProject && (
            <button type="button" className="btn btn--primary" onClick={onCreateProject}>
              去创建项目
            </button>
          )}
        />
      </div>
    );
  }

  return (
    <div className="field-group">
      <label>{label} <b>*</b></label>
      <select className="styled-select" value={value || ""} onChange={(e) => onChange(e.target.value)} disabled={locked}>
        <option value="">请选择项目</option>
        {projects.map((project) => (
          <option key={project.id} value={project.id}>
            {project.name}
          </option>
        ))}
      </select>
      {locked && <span className="field-hint">已根据当前项目自动选中</span>}
    </div>
  );
}
