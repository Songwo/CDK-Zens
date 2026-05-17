import { useEffect, useMemo, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { adminApi } from "../../lib/api";
import CampaignCreateModal from "../../components/CampaignCreateModal";
import CdkImportModal from "../../components/CdkImportModal";
import { notifyOnboardingRefresh } from "../../components/DistributionGuide";
import { ConfirmDialog, DataTable, EmptyState, FilterSelect, PageHeader, ProgressBar, SearchInput, StatusBadge, Toast } from "../../components/ui";

function items(res) { return res?.items || res || []; }

function formatTime(iso) {
  if (!iso) return "--";
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString("zh-CN");
}

export default function CampaignListPage() {
  const location = useLocation();
  const navigate = useNavigate();
  const [campaigns, setCampaigns] = useState([]);
  const [projects, setProjects] = useState([]);
  const [loading, setLoading] = useState(true);
  const [projectsLoading, setProjectsLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("all");
  const [projectId, setProjectId] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [defaultProjectId, setDefaultProjectId] = useState("");
  const [lockedProjectId, setLockedProjectId] = useState("");
  const [importTarget, setImportTarget] = useState(null);
  const [confirm, setConfirm] = useState(null);
  const [toast, setToast] = useState(null);
  const mode = location.pathname.endsWith("/rules") ? "rules" : location.pathname.endsWith("/archive") ? "archive" : "list";

  async function reloadProjects() {
    setProjectsLoading(true);
    try {
      setProjects(items(await adminApi.listProjects({ pageSize: 500 })));
    } catch (err) {
      setToast({ type: "error", message: err.message });
    } finally {
      setProjectsLoading(false);
    }
  }

  async function reloadCampaigns() {
    setLoading(true);
    try {
      setCampaigns(items(await adminApi.listCampaigns({ keyword: search, status, projectId, pageSize: 500 })));
    } catch (err) {
      setToast({ type: "error", message: err.message });
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { reloadProjects(); }, []);
  useEffect(() => { reloadCampaigns(); }, [search, status, projectId]);

  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const open = params.get("open") === "create" || location.pathname.endsWith("/create");
    if (!open) return;
    const routeProjectId = params.get("projectId") || "";
    setDefaultProjectId(routeProjectId);
    setLockedProjectId(routeProjectId);
    setCreateOpen(true);
  }, [location.pathname, location.search]);

  const rows = useMemo(() => {
    if (mode !== "archive") return campaigns;
    return campaigns.filter((c) => ["ended", "exhausted", "paused", "disabled"].includes(c.status));
  }, [campaigns, mode]);

  function openCreate(project = null) {
    setDefaultProjectId(project?.id || projectId || "");
    setLockedProjectId("");
    setCreateOpen(true);
  }

  async function handleCreated() {
    setCreateOpen(false);
    setToast({ message: "活动创建成功" });
    await reloadCampaigns();
    notifyOnboardingRefresh();
    const returnTo = new URLSearchParams(location.search).get("returnTo");
    if (returnTo) {
      navigate(returnTo, { replace: true });
      return;
    }
    if (location.pathname.endsWith("/create") || location.search) {
      navigate("/admin/campaigns", { replace: true });
    }
  }

  async function handleImported(res, campaign) {
    setImportTarget(null);
    const imported = res.imported || res.successCount || 0;
    const usedElsewhere = res.usedElsewhere || 0;
    const dupes = res.duplicates || 0;
    let msg = `成功导入 ${imported} 个 CDK 到活动【${campaign?.name || "未知活动"}】`;
    if (usedElsewhere > 0 || dupes > 0) {
      const parts = [];
      if (usedElsewhere > 0) parts.push(`${usedElsewhere} 个被其他活动占用`);
      if (dupes > 0) parts.push(`${dupes} 个本批重复`);
      msg += `，跳过 ${parts.join("、")}`;
    }
    setToast({ message: msg, type: usedElsewhere > 0 ? "error" : "success" });
    await reloadCampaigns();
    notifyOnboardingRefresh();
  }

  async function runAction(action, item) {
    try {
      if (action === "pause") await adminApi.pauseCampaign(item.id);
      if (action === "resume") await adminApi.resumeCampaign(item.id);
      if (action === "end") await adminApi.endCampaign(item.id);
      if (action === "delete") await adminApi.deleteCampaign(item.id);
      setToast({ message: "操作成功" });
      await reloadCampaigns();
      notifyOnboardingRefresh();
    } catch (err) {
      setToast({ type: "error", message: err.message });
    } finally {
      setConfirm(null);
    }
  }

  return (
    <div className="admin-page">
      {toast && <Toast {...toast} onClose={() => setToast(null)} />}
      <PageHeader
        eyebrow="Campaigns"
        title={mode === "rules" ? "领取规则" : mode === "archive" ? "归档活动" : "活动列表"}
        description={mode === "rules" ? "集中查看活动的重复领取、IP、设备和验证码策略。" : mode === "archive" ? "查看已结束、库存耗尽或已暂停的历史活动。" : "管理 CDK 活动、库存、状态和领取记录。"}
        actions={<button className="btn btn--primary" onClick={() => openCreate()}>新建活动</button>}
      />

      {!projectsLoading && projects.length === 0 && (
        <EmptyState
          title="你还没有项目。请先创建项目，用来归类活动和分发节点。"
          action={<button className="btn btn--primary" onClick={() => navigate(`/admin/projects?open=create&returnTo=${encodeURIComponent("/admin/campaigns?open=create")}`)}>创建项目</button>}
        />
      )}

      {mode === "rules" && (
        <section className="panel">
          <div className="panel__header"><div><h2>规则总览</h2><p>这些字段会在公开领取接口中参与校验或展示。</p></div></div>
          <DataTable rows={campaigns} pageSize={8} emptyText="暂无活动。创建活动后才能配置领取规则。" columns={[
            { key: "name", title: "活动" },
            { key: "projectName", title: "所属项目" },
            { key: "allowRepeat", title: "允许重复", render: (r) => r.allowRepeat ? "允许" : "不允许" },
            { key: "perUserLimit", title: "单用户限制" },
            { key: "perIPLimit", title: "单 IP 限制", render: (r) => r.perIPLimit || "默认" },
            { key: "perDeviceLimit", title: "单设备限制", render: (r) => r.perDeviceLimit || "默认" },
            { key: "requireCaptchaDefault", title: "默认验证码", render: (r) => <StatusBadge status={r.requireCaptchaDefault ? "active" : "paused"} /> },
          ]} />
        </section>
      )}

      <div className="toolbar panel-toolbar">
        <FilterSelect value={projectId} onChange={setProjectId}>
          <option value="">全部项目</option>
          {projects.map((project) => <option key={project.id} value={project.id}>{project.name}</option>)}
        </FilterSelect>
        <FilterSelect value={status} onChange={setStatus}>
          <option value="all">全部状态</option><option value="active">进行中</option><option value="disabled">已暂停</option><option value="paused">已暂停</option><option value="draft">未开始</option><option value="ended">已结束</option><option value="exhausted">库存耗尽</option>
        </FilterSelect>
        <SearchInput value={search} onChange={setSearch} placeholder="搜索活动名称、活动 ID、项目名称、短链接" />
      </div>

      <section className="panel">
        <DataTable loading={loading} rows={rows} pageSize={10} emptyText="暂无活动。创建活动后才能导入 CDK 并生成领取链接。" columns={[
          { key: "name", title: "活动", render: (r) => <><strong>{r.name}</strong><small>{r.id}<br />{r.projectCode}</small></> },
          { key: "projectName", title: "所属项目", render: (r) => r.projectName || "--" },
          { key: "status", title: "状态", render: (r) => <StatusBadge status={r.status} /> },
          { key: "stock", title: "库存进度", render: (r) => <ProgressBar claimed={r.claimedCount} total={r.totalStock} remaining={r.remaining} /> },
          { key: "time", title: "时间范围", render: (r) => <span>{formatTime(r.startTime)}<br /><small>{formatTime(r.endTime)}</small></span> },
          { key: "nodeCount", title: "绑定节点", render: (r) => r.nodeCount || 0 },
          { key: "actions", title: "操作", render: (r) => <div className="row-actions">
            {(r.cdkPoolLocked || (r.totalStock || 0) > 0)
              ? <button className="btn btn--text" onClick={() => navigate(`/admin/cdks?campaignId=${r.id}&projectId=${r.projectId || ""}`)}>查看 CDK</button>
              : <button className="btn btn--text" onClick={() => setImportTarget(r)}>导入 CDK</button>}
            {r.enabled ? <button className="btn btn--text" onClick={() => setConfirm({ action: "pause", item: r })}>暂停</button> : <button className="btn btn--text" onClick={() => runAction("resume", r)}>恢复</button>}
            <button className="btn btn--text" onClick={() => setConfirm({ action: "end", item: r })}>结束</button>
            <button className="btn btn--text danger" onClick={() => setConfirm({ action: "delete", item: r })}>删除</button>
          </div> },
        ]} />
      </section>

      <CampaignCreateModal
        open={createOpen}
        projects={projects}
        projectsLoading={projectsLoading}
        defaultProjectId={defaultProjectId}
        lockedProjectId={lockedProjectId}
        onCancel={() => setCreateOpen(false)}
        onCreateProject={() => navigate(`/admin/projects?open=create&returnTo=${encodeURIComponent("/admin/campaigns?open=create")}`)}
        onSuccess={handleCreated}
        onError={(message) => setToast({ type: "error", message })}
      />

      <CdkImportModal
        open={!!importTarget}
        projects={projects}
        campaigns={campaigns}
        initialProjectId={importTarget?.projectId}
        initialCampaignId={importTarget?.id}
        lockSelection
        onCancel={() => setImportTarget(null)}
        onCreateProject={() => navigate(`/admin/projects?open=create&returnTo=${encodeURIComponent("/admin/campaigns?open=create")}`)}
        onSuccess={handleImported}
        onError={(message) => setToast({ type: "error", message })}
      />

      <ConfirmDialog
        open={!!confirm}
        danger
        title={confirm?.action === "delete" ? "删除活动" : "暂停活动"}
        description={confirm?.action === "delete" ? "删除后活动、绑定节点和领取入口会被移除。" : "暂停后该活动将拒绝新的领取请求。"}
        confirmText="确认"
        onCancel={() => setConfirm(null)}
        onConfirm={() => runAction(confirm.action, confirm.item)}
      />
    </div>
  );
}
