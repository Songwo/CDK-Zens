import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { adminApi } from "../../lib/api";
import { CopyButton, EmptyState, PageHeader, ProgressBar, StatCard, StatusBadge, Toast } from "../../components/ui";

function formatTime(iso) {
  if (!iso) return "--";
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString("zh-CN");
}

function effectiveClaimStatus(node) {
  if (!node) return "";
  if (node.status !== "active") return node.status;
  return node.campaignStatus || node.status;
}

export default function NodeDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [node, setNode] = useState(null);
  const [campaign, setCampaign] = useState(null);
  const [records, setRecords] = useState([]);
  const [loading, setLoading] = useState(true);
  const [toast, setToast] = useState(null);

  async function reload() {
    setLoading(true);
    try {
      const n = await adminApi.getNode(id);
      setNode(n);
      if (n.campaignId) {
        const [c, r] = await Promise.all([adminApi.getCampaign(n.campaignId), adminApi.getCampaignRecords(n.campaignId).catch(() => [])]);
        setCampaign(c);
        setRecords((r || []).filter((item) => !item.nodeId || item.nodeId === n.id).slice(0, 10));
      }
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { reload(); }, [id]);

  async function toggle() {
    try {
      if (node.status === "active") await adminApi.pauseNode(node.id);
      else await adminApi.resumeNode(node.id);
      setToast({ message: "节点状态已更新" });
      reload();
    } catch (err) {
      setToast({ type: "error", message: err.message });
    }
  }

  if (loading) return <div className="admin-page"><EmptyState title="加载中" /></div>;
  if (!node) return <div className="admin-page"><EmptyState title="节点不存在" /></div>;

  const url = window.location.origin + node.claimUrl;
  return (
    <div className="admin-page">
      {toast && <Toast {...toast} onClose={() => setToast(null)} />}
      <PageHeader
        eyebrow="Node Detail"
        title={node.name}
        description={`节点 ID：${node.id}，Slug：${node.slug}`}
        actions={<><button className="btn btn--secondary" onClick={() => navigate("/admin/nodes")}>返回</button><CopyButton value={url}>复制领取链接</CopyButton><button className="btn btn--primary" onClick={toggle}>{node.status === "active" ? "暂停节点" : "恢复节点"}</button></>}
      />
      <section className="stats-grid">
        <StatCard label="今日访问" value={node.visits} hint="当前为累计访问，可扩展日维度" />
        <StatCard label="累计领取" value={node.claims} hint="节点领取总量" tone="gold" />
        <StatCard label="转化率" value={`${node.conversion || 0}%`} hint="领取 / 访问" tone="success" />
        <StatCard label="剩余库存" value={node.remaining} hint={node.campaignName} />
      </section>
      <section className="dashboard-grid">
        <div className="panel panel--wide">
          <div className="panel__header"><h2>节点基础信息</h2><StatusBadge status={effectiveClaimStatus(node)} /></div>
          <div className="detail-lines">
            <p><b>公开链接</b><span>{url}</span></p>
            <p><b>绑定活动</b><span>{node.campaignName}</span></p>
            <p><b>标题</b><span>{node.title}</span></p>
            <p><b>描述</b><span>{node.description || "--"}</span></p>
            <p><b>按钮文案</b><span>{node.buttonText}</span></p>
            <p><b>最后访问</b><span>{formatTime(node.lastVisitedAt)}</span></p>
          </div>
          {campaign && <ProgressBar claimed={campaign.claimedCount} total={campaign.totalStock} remaining={campaign.remaining} />}
        </div>
        <div className="panel">
          <div className="panel__header"><h2>访问策略</h2></div>
          <div className="tag-list">
            <span>{node.showStock ? "显示库存" : "隐藏库存"}</span>
            <span>{node.showEndTime ? "显示结束时间" : "隐藏结束时间"}</span>
            <span>{node.ipLimitEnabled ? "限制 IP" : "不限制 IP"}</span>
            <span>{node.deviceLimitEnabled ? "限制设备" : "不限制设备"}</span>
            <span>{node.requireCaptcha ? "验证码开启" : "验证码关闭"}</span>
          </div>
        </div>
      </section>
      <section className="panel">
        <div className="panel__header"><h2>最近领取记录</h2></div>
        {records.length === 0 ? <EmptyState title="暂无领取记录" /> : records.map((r) => <div className="record-row" key={r.id}><code>{r.rewardContent}</code><span>{r.ip}</span><small>{formatTime(r.createdAt)}</small></div>)}
      </section>
    </div>
  );
}
