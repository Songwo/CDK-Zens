import { useEffect, useMemo, useState } from "react";
import { adminApi } from "../lib/api";
import ProjectSelect from "./ProjectSelect";
import { FilterSelect, Modal } from "./ui";

function parseCodes(text) {
  const lines = text.split(/\r?\n/).map((v) => v.trim());
  const nonEmpty = lines.filter(Boolean);
  const unique = new Set(nonEmpty);
  return { lines, nonEmpty, unique, duplicates: nonEmpty.length - unique.size, codes: [...unique] };
}

export default function CdkImportModal({
  open,
  projects = [],
  campaigns = [],
  projectsLoading = false,
  initialProjectId = "",
  initialCampaignId = "",
  onCancel,
  onSuccess,
  onError,
  onCreateProject,
}) {
  const [projectId, setProjectId] = useState("");
  const [campaignId, setCampaignId] = useState("");
  const [text, setText] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const filteredCampaigns = useMemo(() => {
    if (!projectId) return campaigns;
    return campaigns.filter((campaign) => campaign.projectId === projectId);
  }, [campaigns, projectId]);

  useEffect(() => {
    if (!open) return;
    const selectedCampaign = campaigns.find((campaign) => campaign.id === initialCampaignId);
    const nextProjectId = initialProjectId || selectedCampaign?.projectId || (projects.length === 1 ? projects[0].id : "");
    const candidates = nextProjectId ? campaigns.filter((campaign) => campaign.projectId === nextProjectId) : campaigns;
    const nextCampaignId = initialCampaignId || candidates[0]?.id || "";
    setProjectId(nextProjectId);
    setCampaignId(nextCampaignId);
    setText("");
  }, [open, initialProjectId, initialCampaignId, projects, campaigns]);

  const preview = useMemo(() => parseCodes(text), [text]);
  const selectedCampaign = campaigns.find((campaign) => campaign.id === campaignId);

  function changeProject(nextProjectId) {
    setProjectId(nextProjectId);
    const nextCampaign = campaigns.find((campaign) => campaign.projectId === nextProjectId);
    setCampaignId(nextCampaign?.id || "");
  }

  async function submit(e) {
    e.preventDefault();
    if (submitting) return;
    if (!projectId) return onError?.("请选择项目");
    if (!campaignId) return onError?.("请选择活动");
    if (!preview.codes.length) return onError?.("CDK 不能为空");
    setSubmitting(true);
    try {
      const res = await adminApi.importCDKs(projectId, campaignId, preview.codes);
      onSuccess?.(res, selectedCampaign);
      setText("");
    } catch (err) {
      onError?.(err.message || "CDK 导入失败");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Modal open={open} title="批量导入 CDK" onCancel={submitting ? undefined : onCancel}>
      <form className="modal-form" onSubmit={submit}>
        <ProjectSelect
          projects={projects}
          loading={projectsLoading}
          value={projectId}
          onChange={changeProject}
          onCreateProject={onCreateProject}
        />
        <div className="field-group">
          <label>所属活动 <b>*</b></label>
          {!campaigns.length ? (
            <div className="empty-panel empty-panel--compact">暂无活动。创建活动后才能导入 CDK 并生成领取链接。</div>
          ) : !filteredCampaigns.length ? (
            <div className="empty-panel empty-panel--compact">该项目下暂无活动，请先创建活动。</div>
          ) : (
            <FilterSelect value={campaignId} onChange={setCampaignId}>
              <option value="">选择活动</option>
              {filteredCampaigns.map((campaign) => (
                <option key={campaign.id} value={campaign.id}>{campaign.name}</option>
              ))}
            </FilterSelect>
          )}
        </div>
        <textarea className="styled-textarea mono" rows="10" value={text} onChange={(e) => setText(e.target.value)} placeholder="每行一个 CDK" />
        <div className="empty-panel empty-panel--compact">
          总行数 {preview.lines.length}，有效 {preview.nonEmpty.length}，去重后 {preview.codes.length}，本批重复 {preview.duplicates}
        </div>
        <div className="modal-footer">
          <button type="button" className="modal-btn modal-btn--cancel" onClick={onCancel} disabled={submitting}>取消</button>
          <button className="modal-btn modal-btn--primary" disabled={submitting || !projectId || !campaignId || !preview.codes.length}>{submitting ? "导入中..." : "导入"}</button>
        </div>
      </form>
    </Modal>
  );
}
