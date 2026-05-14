import { useEffect, useMemo, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { adminApi } from "../../lib/api";
import { notifyOnboardingRefresh } from "../../components/DistributionGuide";
import { ConfirmDialog, CopyButton, DataTable, Modal, PageHeader, ProgressBar, StatusBadge, Toast } from "../../components/ui";

const SLUG_RE = /^[a-zA-Z0-9_-]+$/;
const EMPTY_NODE = { name: "", slug: "", campaignId: "", title: "", description: "", buttonText: "立即领取", limit: 0, showStock: true, showEndTime: true, requireCaptcha: false, ipLimitEnabled: false, deviceLimitEnabled: true };

function formatTime(iso) {
  if (!iso) return "--";
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString("zh-CN");
}

function effectiveClaimStatus(node) {
  if (node.status !== "active") return node.status;
  return node.campaignStatus || node.status;
}

export default function NodeListPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const [nodes, setNodes] = useState([]);
  const [campaigns, setCampaigns] = useState([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("all");
  const [form, setForm] = useState(null);
  const [confirm, setConfirm] = useState(null);
  const [toast, setToast] = useState(null);
  const [submitting, setSubmitting] = useState(false);
  const stockCampaigns = useMemo(() => campaigns.filter((c) => Number(c.remaining ?? c.remainingCount ?? c.totalStock ?? 0) > 0), [campaigns]);

  async function reload() {
    setLoading(true);
    try {
      const [nodeData, campaignData] = await Promise.all([adminApi.listNodes(), adminApi.listCampaigns()]);
      setNodes(nodeData.items || nodeData || []);
      setCampaigns(campaignData.items || campaignData || []);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { reload(); }, []);

  useEffect(() => {
    if (location.pathname.endsWith("/create")) {
      const params = new URLSearchParams(location.search);
      const requested = params.get("campaignId");
      const candidate = stockCampaigns.find((c) => c.id === requested) || stockCampaigns[0];
      if (!campaigns.length) {
        setToast({ type: "error", message: "请先创建活动" });
        return;
      }
      if (!candidate) {
        setToast({ type: "error", message: "请先导入 CDK" });
        return;
      }
      setForm({ ...EMPTY_NODE, campaignId: candidate.id });
    }
  }, [location.pathname, location.search, campaigns, stockCampaigns]);

  const rows = useMemo(() => {
    const q = search.trim().toLowerCase();
    return nodes.filter((n) => {
      if (status !== "all" && n.status !== status) return false;
      if (!q) return true;
      return [n.name, n.slug, n.campaignName, n.description].some((v) => String(v || "").toLowerCase().includes(q));
    });
  }, [nodes, search, status]);

  async function createNode(e) {
    e.preventDefault();
    if (submitting) return;
    if (!form.name.trim() || !form.slug.trim() || !form.campaignId) return setToast({ type: "error", message: "节点名称、slug 和绑定活动不能为空" });
    if (!SLUG_RE.test(form.slug)) return setToast({ type: "error", message: "slug 只能包含字母、数字、短横线和下划线" });
    setSubmitting(true);
    try {
      await adminApi.createNode({ ...form, limit: Number(form.limit) || 0 });
      setForm(null);
      setToast({ message: "节点创建成功" });
      await reload();
      notifyOnboardingRefresh();
      const returnTo = new URLSearchParams(location.search).get("returnTo");
      if (returnTo) navigate(returnTo, { replace: true });
      else if (location.pathname.endsWith("/create")) navigate("/admin/nodes", { replace: true });
    } catch (err) {
      setToast({ type: "error", message: err.message });
    } finally {
      setSubmitting(false);
    }
  }

  async function runAction(action, node) {
    try {
      if (action === "pause") await adminApi.pauseNode(node.id);
      if (action === "resume") await adminApi.resumeNode(node.id);
      if (action === "delete") await adminApi.deleteNode(node.id);
      setToast({ message: "操作成功" });
      reload();
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
        eyebrow="Distribution Network"
        title="分发网络"
        description="管理多个公开领取入口，每个节点都可以绑定独立活动、配置展示和限制规则。"
        actions={<button className="btn btn--primary" onClick={() => {
          if (!campaigns.length) return setToast({ type: "error", message: "请先创建活动" });
          if (!stockCampaigns.length) return setToast({ type: "error", message: "请先导入 CDK" });
          setForm({ ...EMPTY_NODE, campaignId: stockCampaigns[0]?.id || "" });
        }}>新建节点</button>}
      />
      <div className="toolbar panel-toolbar">
        <select className="styled-select" value={status} onChange={(e) => setStatus(e.target.value)}><option value="all">全部状态</option><option value="active">进行中</option><option value="paused">已暂停</option></select>
        <input className="search-input" placeholder="搜索节点名称、slug、活动" value={search} onChange={(e) => setSearch(e.target.value)} />
      </div>
      <section className="panel">
        {!loading && !nodes.length && <div style={{ marginBottom: 12 }}><div className="empty-panel empty-panel--compact">创建分发节点后，用户才能通过公开链接领取 CDK。</div></div>}
        <DataTable loading={loading} rows={rows} pageSize={10} columns={[
          { key: "name", title: "节点", render: (r) => <><strong>{r.name}</strong><small>{r.slug}<br />{window.location.origin + r.claimUrl}</small></> },
          { key: "campaign", title: "绑定活动", render: (r) => <><span>{r.campaignName || "--"}</span><br /><StatusBadge status={r.campaignStatus} /></> },
          { key: "status", title: "领取状态", render: (r) => <><StatusBadge status={effectiveClaimStatus(r)} /><small>节点：{r.status === "active" ? "已启用" : "已暂停"}</small></> },
          { key: "traffic", title: "访问 / 领取 / 转化", render: (r) => <span>{r.visits} / {r.claims} / {r.conversion || 0}%</span> },
          { key: "stock", title: "库存", render: (r) => <ProgressBar claimed={r.claims} total={(r.claims || 0) + (r.remaining || 0)} remaining={r.remaining} /> },
          { key: "time", title: "时间", render: (r) => <span>{formatTime(r.createdAt)}<br /><small>{formatTime(r.lastVisitedAt)}</small></span> },
          { key: "actions", title: "操作", render: (r) => <div className="row-actions">
            <button className="btn btn--text" onClick={() => navigate(`/admin/nodes/${r.id}`)}>查看</button>
            <CopyButton value={window.location.origin + r.claimUrl}>复制链接</CopyButton>
            {r.status === "active" ? <button className="btn btn--text" onClick={() => setConfirm({ action: "pause", item: r })}>暂停</button> : <button className="btn btn--text" onClick={() => runAction("resume", r)}>恢复</button>}
            <button className="btn btn--text danger" onClick={() => setConfirm({ action: "delete", item: r })}>删除</button>
          </div> },
        ]} />
      </section>

      {form && (
        <Modal open={!!form} title="新建分发节点" onCancel={submitting ? undefined : () => setForm(null)}>
        <form className="modal-form" onSubmit={createNode}>
          <select className="styled-select" value={form.campaignId} onChange={(e) => setForm({ ...form, campaignId: e.target.value })}>
            <option value="">选择绑定活动</option>
            {stockCampaigns.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
          </select>
          <input className="styled-input" placeholder="节点名称" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
          <input className="styled-input" placeholder="节点 slug，例如 miukey-main" value={form.slug} onChange={(e) => setForm({ ...form, slug: e.target.value })} />
          <input className="styled-input" placeholder="领取页标题" value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} />
          <textarea className="styled-textarea" placeholder="领取页描述" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
          <div className="form-row-2"><input className="styled-input" placeholder="按钮文案" value={form.buttonText} onChange={(e) => setForm({ ...form, buttonText: e.target.value })} /><input className="styled-input" type="number" min="0" placeholder="领取上限，0 为不限" value={form.limit} onChange={(e) => setForm({ ...form, limit: e.target.value })} /></div>
          <div className="switch-grid">
            {["showStock:显示库存", "showEndTime:显示结束时间", "requireCaptcha:需要验证码", "ipLimitEnabled:限制同一 IP", "deviceLimitEnabled:限制同一设备"].map((item) => {
              const [key, label] = item.split(":");
              return <label className="switch-label" key={key}><input type="checkbox" checked={!!form[key]} onChange={(e) => setForm({ ...form, [key]: e.target.checked })} />{label}</label>;
            })}
          </div>
          <div className="modal-footer"><button type="button" className="modal-btn modal-btn--cancel" onClick={() => setForm(null)} disabled={submitting}>取消</button><button className="modal-btn modal-btn--primary" disabled={submitting}>{submitting ? "创建中..." : "创建"}</button></div>
        </form>
        </Modal>
      )}

      <ConfirmDialog
        open={!!confirm}
        danger
        title={confirm?.action === "delete" ? "删除节点" : "暂停节点"}
        description={confirm?.action === "delete" ? "删除后公开领取链接将不可访问。" : "暂停后该节点会立即拒绝领取。"}
        confirmText="确认"
        onCancel={() => setConfirm(null)}
        onConfirm={() => runAction(confirm.action, confirm.item)}
      />
    </div>
  );
}
