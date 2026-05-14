import { useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { adminApi } from "../../lib/api";
import { notifyOnboardingRefresh } from "../../components/DistributionGuide";
import { ConfirmDialog, DataTable, Modal, PageHeader, SearchInput, StatCard, StatusBadge, Toast } from "../../components/ui";

function items(res) { return res?.items || res || []; }
function time(iso) { const d = new Date(iso); return iso && !Number.isNaN(d.getTime()) ? d.toLocaleString("zh-CN") : "--"; }

export default function ProjectListPage() {
  const location = useLocation();
  const navigate = useNavigate();
  const [rows, setRows] = useState([]);
  const [keyword, setKeyword] = useState("");
  const [loading, setLoading] = useState(true);
  const [form, setForm] = useState(null);
  const [detail, setDetail] = useState(null);
  const [confirm, setConfirm] = useState(null);
  const [toast, setToast] = useState(null);
  const [submitting, setSubmitting] = useState(false);

  async function reload() {
    setLoading(true);
    try { setRows(items(await adminApi.listProjects({ keyword, pageSize: 200 }))); }
    catch (err) { setToast({ type: "error", message: err.message }); }
    finally { setLoading(false); }
  }
  useEffect(() => { reload(); }, [keyword]);

  useEffect(() => {
    const params = new URLSearchParams(location.search);
    if (params.get("open") === "create") setForm({ name: "", description: "" });
  }, [location.search]);

  async function save(e) {
    e.preventDefault();
    if (submitting) return;
    if (!form?.name?.trim()) return setToast({ type: "error", message: "项目名称不能为空" });
    setSubmitting(true);
    try {
      if (form.id) await adminApi.updateProject(form.id, form);
      else await adminApi.createProject(form);
      setForm(null);
      setToast({ message: "项目已保存" });
      await reload();
      notifyOnboardingRefresh();
      const params = new URLSearchParams(location.search);
      if (params.get("open") === "create") {
        navigate(params.get("returnTo") || "/admin/projects", { replace: true });
      }
    } catch (err) {
      setToast({ type: "error", message: err.message });
    } finally {
      setSubmitting(false);
    }
  }

  async function run(action, row) {
    try {
      if (action === "archive") await adminApi.archiveProject(row.id);
      if (action === "delete") await adminApi.deleteProject(row.id);
      setToast({ message: "操作成功" });
      reload();
    } catch (err) {
      setToast({ type: "error", message: err.message });
    } finally {
      setConfirm(null);
    }
  }

  async function openDetail(row) {
    try { setDetail(await adminApi.getProject(row.id)); }
    catch (err) { setToast({ type: "error", message: err.message }); }
  }

  return (
    <div className="admin-page">
      {toast && <Toast {...toast} onClose={() => setToast(null)} />}
      <PageHeader eyebrow="Projects" title="项目管理" description="项目用于组织活动和分发节点，可归档、统计和绑定活动。" actions={<button className="btn btn--primary" onClick={() => setForm({ name: "", description: "" })}>新建项目</button>} />
      <section className="stats-grid">
        <StatCard label="项目总数" value={rows.length} />
        <StatCard label="活跃项目" value={rows.filter((r) => r.status === "active").length} tone="success" />
        <StatCard label="活动数量" value={rows.reduce((n, r) => n + Number(r.campaignCount || 0), 0)} />
        <StatCard label="节点数量" value={rows.reduce((n, r) => n + Number(r.nodeCount || 0), 0)} />
      </section>
      <div className="toolbar panel-toolbar"><SearchInput value={keyword} onChange={setKeyword} placeholder="搜索项目名称 / 描述" /></div>
      <section className="panel">
        <DataTable loading={loading} rows={rows} pageSize={12} columns={[
          { key: "name", title: "项目", render: (r) => <><strong>{r.name}</strong><small>{r.description || r.id}</small></> },
          { key: "status", title: "状态", render: (r) => <StatusBadge status={r.status} /> },
          { key: "campaignCount", title: "活动" },
          { key: "nodeCount", title: "节点" },
          { key: "updatedAt", title: "更新时间", render: (r) => time(r.updatedAt) },
          { key: "actions", title: "操作", render: (r) => <div className="row-actions"><button className="btn btn--text" onClick={() => openDetail(r)}>详情</button><button className="btn btn--text" onClick={() => setForm(r)}>编辑</button><button className="btn btn--text" onClick={() => setConfirm({ action: "archive", row: r })}>归档</button><button className="btn btn--text danger" onClick={() => setConfirm({ action: "delete", row: r })}>删除</button></div> },
        ]} />
      </section>
      <Modal open={!!form} title={form?.id ? "编辑项目" : "新建项目"} onCancel={submitting ? undefined : () => setForm(null)}>
        <form className="modal-form" onSubmit={save}>
          <input className="styled-input" value={form?.name || ""} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="项目名称" />
          <textarea className="styled-textarea" value={form?.description || ""} onChange={(e) => setForm({ ...form, description: e.target.value })} placeholder="项目描述" />
          <div className="modal-footer"><button type="button" className="modal-btn modal-btn--cancel" onClick={() => setForm(null)} disabled={submitting}>取消</button><button className="modal-btn modal-btn--primary" disabled={submitting}>{submitting ? "保存中..." : "保存"}</button></div>
        </form>
      </Modal>
      <Modal open={!!detail} title="项目详情" onCancel={() => setDetail(null)}>
        <div className="modal-body">
          <p><b>活动数量：</b>{detail?.campaigns?.length || 0}</p>
          <p><b>节点数量：</b>{detail?.nodes?.length || 0}</p>
          <pre>{JSON.stringify(detail, null, 2)}</pre>
        </div>
      </Modal>
      <ConfirmDialog open={!!confirm} danger={confirm?.action === "delete"} title={confirm?.action === "delete" ? "删除项目" : "归档项目"} description={confirm?.action === "delete" ? "项目下存在活动或节点时无法删除，确认尝试删除？" : "归档后项目会进入 archived 状态。"} onCancel={() => setConfirm(null)} onConfirm={() => run(confirm.action, confirm.row)} />
    </div>
  );
}
