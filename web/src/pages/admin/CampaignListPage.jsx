import { useEffect, useMemo, useState } from "react";
import { useLocation } from "react-router-dom";
import { adminApi } from "../../lib/api";
import { ConfirmDialog, DataTable, Modal, PageHeader, ProgressBar, StatusBadge, Toast } from "../../components/ui";

const EMPTY_FORM = { name: "", description: "", startTime: "", endTime: "", enabled: true, perUserLimit: 1, rewardListText: "" };

function toISO(value) {
  return value ? new Date(value).toISOString() : "";
}

function formatTime(iso) {
  if (!iso) return "--";
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString("zh-CN");
}

export default function CampaignListPage() {
  const location = useLocation();
  const [campaigns, setCampaigns] = useState([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("all");
  const [form, setForm] = useState(null);
  const [importTarget, setImportTarget] = useState(null);
  const [importText, setImportText] = useState("");
  const [confirm, setConfirm] = useState(null);
  const [toast, setToast] = useState(null);
  const mode = location.pathname.endsWith("/create") ? "create" : location.pathname.endsWith("/rules") ? "rules" : location.pathname.endsWith("/archive") ? "archive" : "list";

  async function reload() {
    setLoading(true);
    try {
      const res = await adminApi.listCampaigns();
      setCampaigns(res.items || res || []);
    } finally { setLoading(false); }
  }

  useEffect(() => { reload(); }, []);

  useEffect(() => {
    if (location.pathname.endsWith("/create")) setForm(EMPTY_FORM);
    if (location.pathname.endsWith("/archive")) setStatus("all");
  }, [location.pathname]);

  const rows = useMemo(() => {
    const q = search.trim().toLowerCase();
    return campaigns.filter((c) => {
      if (mode === "archive" && !["ended", "exhausted", "paused"].includes(c.status)) return false;
      if (status !== "all" && c.status !== status) return false;
      if (!q) return true;
      return [c.name, c.projectCode, c.claimUrl].some((v) => String(v || "").toLowerCase().includes(q));
    });
  }, [campaigns, search, status]);

  async function createCampaign(e) {
    e.preventDefault();
    if (!form.name.trim()) return setToast({ type: "error", message: "活动名称不能为空" });
    const items = form.rewardListText.split(/\r?\n|,|;|，|；/).map((v) => v.trim()).filter(Boolean);
    if (items.length === 0) return setToast({ type: "error", message: "CDK 不能为空" });
    try {
      await adminApi.createCampaign({
        name: form.name.trim(),
        description: form.description,
        startAt: toISO(form.startTime),
        endAt: toISO(form.endTime),
        rewardType: "cdk_list",
        rewardList: items,
        enabled: form.enabled,
        perUserLimit: Number(form.perUserLimit) || 1,
      });
      setForm(null);
      setToast({ message: "活动创建成功" });
      reload();
    } catch (err) {
      setToast({ type: "error", message: err.message });
    }
  }

  async function importCDK(e) {
    e.preventDefault();
    const items = importText.split(/\r?\n|,|;|，|；/).map((v) => v.trim()).filter(Boolean);
    if (!items.length) return setToast({ type: "error", message: "CDK 不能为空" });
    try {
      const res = await adminApi.importCampaignCDKs(importTarget.id, items);
      setToast({ message: `导入 ${res.imported} 条，重复 ${res.duplicates} 条，无效 ${res.invalid} 条` });
      setImportTarget(null);
      setImportText("");
      reload();
    } catch (err) {
      setToast({ type: "error", message: err.message });
    }
  }

  async function runAction(action, item) {
    try {
      if (action === "pause") await adminApi.pauseCampaign(item.id);
      if (action === "resume") await adminApi.resumeCampaign(item.id);
      if (action === "end") await adminApi.endCampaign(item.id);
      if (action === "delete") await adminApi.deleteCampaign(item.id);
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
        eyebrow="Campaigns"
        title={mode === "create" ? "新建活动" : mode === "rules" ? "领取规则" : mode === "archive" ? "归档活动" : "活动列表"}
        description={mode === "rules" ? "集中查看活动的重复领取、IP、设备和验证码策略。" : mode === "archive" ? "查看已结束、库存耗尽或已暂停的历史活动。" : "管理 CDK 活动、库存、状态和领取记录。"}
        actions={<button className="btn btn--primary" onClick={() => setForm(EMPTY_FORM)}>新建活动</button>}
      />

      {mode === "rules" && (
        <section className="panel">
          <div className="panel__header"><div><h2>规则总览</h2><p>这些字段会在公开领取接口中参与校验或展示。</p></div></div>
          <DataTable rows={campaigns} pageSize={8} columns={[
            { key: "name", title: "活动" },
            { key: "allowRepeat", title: "允许重复", render: (r) => r.allowRepeat ? "允许" : "不允许" },
            { key: "perUserLimit", title: "单用户限制" },
            { key: "perIPLimit", title: "单 IP 限制", render: (r) => r.perIPLimit || "默认" },
            { key: "perDeviceLimit", title: "单设备限制", render: (r) => r.perDeviceLimit || "默认" },
            { key: "requireCaptchaDefault", title: "默认验证码", render: (r) => <StatusBadge status={r.requireCaptchaDefault ? "active" : "paused"} /> },
          ]} />
        </section>
      )}

      <div className="toolbar panel-toolbar">
        <select className="styled-select" value={status} onChange={(e) => setStatus(e.target.value)}>
          <option value="all">全部状态</option><option value="active">进行中</option><option value="disabled">已暂停</option><option value="upcoming">未开始</option><option value="ended">已结束</option><option value="soldout">库存耗尽</option>
        </select>
        <input className="search-input" placeholder="搜索活动名称、ID、短链接" value={search} onChange={(e) => setSearch(e.target.value)} />
      </div>

      <section className="panel">
        <DataTable loading={loading} rows={rows} pageSize={10} columns={[
          { key: "name", title: "活动", render: (r) => <><strong>{r.name}</strong><small>{r.id}<br />{r.projectCode}</small></> },
          { key: "status", title: "状态", render: (r) => <StatusBadge status={r.status} /> },
          { key: "stock", title: "库存进度", render: (r) => <ProgressBar claimed={r.claimedCount} total={r.totalStock} remaining={r.remaining} /> },
          { key: "time", title: "时间范围", render: (r) => <span>{formatTime(r.startTime)}<br /><small>{formatTime(r.endTime)}</small></span> },
          { key: "nodeCount", title: "绑定节点", render: (r) => r.nodeCount || 0 },
          { key: "actions", title: "操作", render: (r) => <div className="row-actions">
            <button className="btn btn--text" onClick={() => setImportTarget(r)}>导入 CDK</button>
            {r.enabled ? <button className="btn btn--text" onClick={() => setConfirm({ action: "pause", item: r })}>暂停</button> : <button className="btn btn--text" onClick={() => runAction("resume", r)}>恢复</button>}
            <button className="btn btn--text" onClick={() => setConfirm({ action: "end", item: r })}>结束</button>
            <button className="btn btn--text danger" onClick={() => setConfirm({ action: "delete", item: r })}>删除</button>
          </div> },
        ]} />
      </section>

      {form && (
        <Modal open={!!form} title="新建活动" onCancel={() => setForm(null)}>
        <form className="modal-form" onSubmit={createCampaign}>
          <input className="styled-input" placeholder="活动名称" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
          <textarea className="styled-textarea" placeholder="活动描述" value={form.description} onChange={(e) => setForm({ ...form, description: e.target.value })} />
          <div className="form-row-2"><input className="styled-input" type="datetime-local" value={form.startTime} onChange={(e) => setForm({ ...form, startTime: e.target.value })} /><input className="styled-input" type="datetime-local" value={form.endTime} onChange={(e) => setForm({ ...form, endTime: e.target.value })} /></div>
          <input className="styled-input" type="number" min="1" value={form.perUserLimit} onChange={(e) => setForm({ ...form, perUserLimit: e.target.value })} />
          <textarea className="styled-textarea mono" rows="7" placeholder="每行一个 CDK" value={form.rewardListText} onChange={(e) => setForm({ ...form, rewardListText: e.target.value })} />
          <label className="switch-label"><input type="checkbox" checked={form.enabled} onChange={(e) => setForm({ ...form, enabled: e.target.checked })} />启用活动</label>
          <div className="modal-footer"><button type="button" className="modal-btn modal-btn--cancel" onClick={() => setForm(null)}>取消</button><button className="modal-btn modal-btn--primary">创建</button></div>
        </form>
        </Modal>
      )}

      {importTarget && (
        <Modal open={!!importTarget} title={`导入 CDK：${importTarget.name}`} onCancel={() => setImportTarget(null)}>
        <form className="modal-form" onSubmit={importCDK}>
          <textarea className="styled-textarea mono" rows="10" placeholder="每行一个 CDK，系统会自动去重" value={importText} onChange={(e) => setImportText(e.target.value)} />
          <div className="modal-footer"><button type="button" className="modal-btn modal-btn--cancel" onClick={() => setImportTarget(null)}>取消</button><button className="modal-btn modal-btn--primary">导入</button></div>
        </form>
        </Modal>
      )}

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
