import { useEffect, useMemo, useState } from "react";
import { adminApi } from "../lib/api";
import ProjectSelect from "./ProjectSelect";
import { Modal } from "./ui";

const EMPTY_FORM = {
  projectId: "",
  name: "",
  description: "",
  startTime: "",
  endTime: "",
  enabled: true,
  perUserLimit: 1,
  requireCaptchaDefault: true,
  importWithCdk: false,
  rewardListText: "",
};

function toISO(value) {
  return value ? new Date(value).toISOString() : "";
}

function parseCodes(text) {
  return text.split(/\r?\n|,|;|，|；/).map((v) => v.trim()).filter(Boolean);
}

export default function CampaignCreateModal({
  open,
  projects = [],
  projectsLoading = false,
  defaultProjectId = "",
  lockedProjectId = "",
  onCancel,
  onSuccess,
  onCreateProject,
  onError,
}) {
  const initialProjectId = lockedProjectId || defaultProjectId || (projects.length === 1 ? projects[0].id : "");
  const [form, setForm] = useState({ ...EMPTY_FORM, projectId: initialProjectId });
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (!open) return;
    const nextProjectId = lockedProjectId || defaultProjectId || (projects.length === 1 ? projects[0].id : "");
    setForm({ ...EMPTY_FORM, projectId: nextProjectId });
  }, [open, lockedProjectId, defaultProjectId, projects]);

  const codes = useMemo(() => parseCodes(form.rewardListText), [form.rewardListText]);

  async function submit(e) {
    e.preventDefault();
    if (submitting) return;
    if (!form.projectId) return onError?.("请选择所属项目");
    if (!form.name.trim()) return onError?.("活动名称不能为空");
    if (form.importWithCdk && codes.length === 0) return onError?.("CDK 不能为空");
    setSubmitting(true);
    try {
      const campaign = await adminApi.createCampaign({
        projectId: form.projectId,
        name: form.name.trim(),
        description: form.description,
        startAt: toISO(form.startTime),
        endAt: toISO(form.endTime),
        enabled: form.enabled,
        perUserLimit: Number(form.perUserLimit) || 1,
        requireCaptchaDefault: form.requireCaptchaDefault,
        rewardType: "cdk_list",
        rewardList: form.importWithCdk ? codes : [],
      });
      onSuccess?.(campaign);
    } catch (err) {
      onError?.(err.message || "活动创建失败");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Modal open={open} title="新建活动" onCancel={submitting ? undefined : onCancel}>
      <form className="modal-form" onSubmit={submit}>
        <ProjectSelect
          projects={projects}
          loading={projectsLoading}
          value={form.projectId}
          onChange={(projectId) => setForm((prev) => ({ ...prev, projectId }))}
          locked={!!lockedProjectId}
          onCreateProject={onCreateProject}
        />
        <input className="styled-input" placeholder="活动名称" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
        <textarea className="styled-textarea" placeholder="活动描述" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
        <div className="form-row-2">
          <input className="styled-input" type="datetime-local" value={form.startTime} onChange={(e) => setForm({ ...form, startTime: e.target.value })} />
          <input className="styled-input" type="datetime-local" value={form.endTime} onChange={(e) => setForm({ ...form, endTime: e.target.value })} />
        </div>
        <input className="styled-input" type="number" min="1" value={form.perUserLimit} onChange={(e) => setForm({ ...form, perUserLimit: e.target.value })} placeholder="每个用户可领取次数" />
        <label className="switch-label"><input type="checkbox" checked={form.importWithCdk} onChange={(e) => setForm({ ...form, importWithCdk: e.target.checked })} />创建活动时同时导入 CDK</label>
        {form.importWithCdk && (
          <>
            <textarea className="styled-textarea mono" rows="7" placeholder="每行一个 CDK" value={form.rewardListText} onChange={(e) => setForm({ ...form, rewardListText: e.target.value })} />
            <div className="empty-panel empty-panel--compact">将导入 {codes.length} 个 CDK</div>
          </>
        )}
        <label className="switch-label"><input type="checkbox" checked={form.requireCaptchaDefault} onChange={(e) => setForm({ ...form, requireCaptchaDefault: e.target.checked })} />默认需要验证码</label>
        <label className="switch-label"><input type="checkbox" checked={form.enabled} onChange={(e) => setForm({ ...form, enabled: e.target.checked })} />启用活动</label>
        <div className="modal-footer">
          <button type="button" className="modal-btn modal-btn--cancel" onClick={onCancel} disabled={submitting}>取消</button>
          <button className="modal-btn modal-btn--primary" disabled={submitting || projectsLoading || !projects.length}>{submitting ? "创建中..." : "创建"}</button>
        </div>
      </form>
    </Modal>
  );
}
