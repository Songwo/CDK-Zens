import { useEffect, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { adminApi } from "../../lib/api";
import Modal from "../../components/Modal";

function formatTime(iso) {
  if (!iso) return "--";
  const d = new Date(iso);
  return isNaN(d.getTime()) ? iso : d.toLocaleString("zh-CN");
}

const STATUS_MAP = {
  active: "正在分发", upcoming: "等待激活", ended: "周期结束", soldout: "额度耗尽", disabled: "已关停",
};

/* 轻量 Toast 组件 */
function Toast({ message, onClose }) {
  useEffect(() => {
    const t = setTimeout(onClose, 2500);
    return () => clearTimeout(t);
  }, []);
  const isErr = message.includes("✗");
  return (
    <div className="cp-toast">
      <div style={{width: 8, height: 8, background: isErr ? "#ef4444" : "var(--cp-brand)", borderRadius: "50%", boxShadow: isErr ? "" : "0 0 8px rgba(244,180,0,0.5)"}} />
      <span>{message.replace(/^[✓✗] /, "")}</span>
    </div>
  );
}

export default function ProjectDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [project, setProject] = useState(null);
  const [records, setRecords] = useState([]);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState("info");
  const [copyOk, setCopyOk] = useState(false);
  const [toast, setToast] = useState("");
  const [disableModalOpen, setDisableModalOpen] = useState(false);

  useEffect(() => {
    Promise.all([
      adminApi.getProject(id),
      adminApi.getRecords(id).catch(() => []),
    ]).then(([proj, recs]) => {
      setProject(proj);
      setRecords(recs || []);
    }).finally(() => setLoading(false));
  }, [id]);

  function copyLink() {
    if (!project) return;
    navigator.clipboard.writeText(window.location.origin + project.claimUrl);
    setCopyOk(true);
    setTimeout(() => setCopyOk(false), 1500);
  }

  async function confirmDisable() {
    try {
      await adminApi.disableProject(id);
      setProject(prev => ({ ...prev, enabled: false, status: "disabled" }));
      setToast("✓ 项目已停用");
    } catch (err) {
      setToast("✗ 操作失败: " + err.message);
    } finally {
      setDisableModalOpen(false);
    }
  }

  function handleDisable() {
    setDisableModalOpen(true);
  }

  if (loading) return <div className="admin-page"><div className="empty-state"><p>正在读取节点矩阵...</p></div></div>;
  if (!project) return <div className="admin-page"><div className="empty-state"><p>节点地址不存在或已从网络移除</p></div></div>;

  const progress = project.totalStock > 0 ? Math.round((project.claimedCount / project.totalStock) * 100) : 0;

  return (
    <div className="admin-page anim-fade-up stagger-1">
      {toast && <Toast message={toast} onClose={() => setToast("")} />}
      
      <header className="page-hero" style={{marginBottom: '32px'}}>
        <button className="btn btn--text" style={{marginLeft: '-12px', marginBottom: '16px'}} onClick={() => navigate("/admin/projects")}>← 返回网络拓扑</button>
        <div style={{display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start'}}>
          <div>
            <h1 className="page-hero__title" style={{marginBottom: '8px'}}>{project.name}</h1>
            <span className="page-hero__subtitle" style={{color: 'var(--cp-muted)'}}>Node ID: {project.projectCode}</span>
          </div>
          <div className="btn-group">
            <button className="btn btn--primary" onClick={() => navigate(`/admin/campaigns?open=create&projectId=${project.id}`)}>创建活动</button>
            <button className="btn btn--secondary" onClick={copyLink} title="复制该项目的公开领取链接，发送给用户即可领取">{copyOk ? "✓ 已复制" : "复制领取链接"}</button>
            {project.enabled && <button className="btn btn--secondary" style={{color: '#dc2626', borderColor: '#fee2e2'}} onClick={handleDisable} title="停用该项目，停用后用户将无法继续领取">停用项目</button>}
          </div>
        </div>
      </header>

      {/* 极简选项卡 */}
      <div className="filter-bar anim-fade-up stagger-2">
        {[["info", "状态矩阵"], ["records", "审计日志"], ["pool", "资源池状态"]].map(([k, l]) => (
          <button key={k} className={`filter-tab ${tab === k ? "filter-tab--active" : ""}`} onClick={() => setTab(k)}>{l}</button>
        ))}
      </div>

      <div className="anim-fade-up stagger-3">
        {tab === "info" && (
          <div className="detail-grid">
            <div className="detail-main">
              <section>
                <div className="section-header"><h2>负载容量指标</h2></div>
                <div style={{display: 'flex', gap: '48px', marginBottom: '24px'}}>
                  <div>
                    <div style={{fontSize: '40px', fontWeight: 700, lineHeight: 1}}>{project.totalStock}</div>
                    <div style={{fontSize: '13px', color: 'var(--cp-muted)', marginTop: '8px'}}>注入总容量</div>
                  </div>
                  <div>
                    <div style={{fontSize: '40px', fontWeight: 700, lineHeight: 1, color: 'var(--cp-brand)'}}>{project.claimedCount}</div>
                    <div style={{fontSize: '13px', color: 'var(--cp-muted)', marginTop: '8px'}}>已释放单位</div>
                  </div>
                  <div>
                    <div style={{fontSize: '40px', fontWeight: 700, lineHeight: 1}}>{project.remaining}</div>
                    <div style={{fontSize: '13px', color: 'var(--cp-muted)', marginTop: '8px'}}>保留量</div>
                  </div>
                </div>
                
                <div className="track-bar" style={{height: '12px', borderRadius: '6px', background: 'var(--cp-surface)', border: '1px solid var(--cp-border)'}}>
                  <div className="track-bar__fill" style={{width: `${progress}%`}} />
                </div>
                <div className="track-meta">
                  <span>释放率 {progress}%</span>
                </div>
              </section>

              <section>
                <div className="section-header"><h2>策略与约束规则</h2></div>
                {project.description && (
                  <div style={{marginBottom: '32px'}}>
                    <h4 style={{fontSize: '14px', marginBottom: '8px', color: 'var(--cp-muted)'}}>公开声明文本</h4>
                    <p style={{fontSize: '15px', lineHeight: 1.6}}>{project.description}</p>
                  </div>
                )}
                {project.rules && (
                  <div>
                    <h4 style={{fontSize: '14px', marginBottom: '8px', color: 'var(--cp-muted)'}}>使用协议与规章</h4>
                    <div style={{background: 'var(--cp-surface)', padding: '20px', borderRadius: '8px', border: '1px solid var(--cp-divider)'}}>
                      <p style={{fontSize: '14px', lineHeight: 1.6, whiteSpace: 'pre-wrap'}}>{project.rules}</p>
                    </div>
                  </div>
                )}
              </section>
            </div>

            <aside className="detail-aside">
              <div className="data-group">
                <div className="data-group__label">网络入口地址</div>
                <div className="link-box" style={{marginTop: '8px'}}>
                  <input type="text" readOnly value={window.location.origin + project.claimUrl} onClick={e => e.target.select()} />
                </div>
              </div>
              <div className="data-group">
                <div className="data-group__label">网关状态</div>
                <div className="data-group__value"><span className={`status-dot status--${project.status}`}>{STATUS_MAP[project.status]}</span></div>
              </div>
              <div className="data-group">
                <div className="data-group__label">资源类型</div>
                <div className="data-group__value">{project.rewardType}</div>
              </div>
              <div className="data-group">
                <div className="data-group__label">频率控制</div>
                <div className="data-group__value">每个终端 {project.perUserLimit} 次限制</div>
              </div>
              <div className="data-group">
                <div className="data-group__label">访问凭证</div>
                <div className="data-group__value">{project.needLogin ? "需要有效身份" : "公开访问"}</div>
              </div>
              <div className="data-group">
                <div className="data-group__label">生命周期</div>
                <div className="data-group__value" style={{fontSize: '14px', lineHeight: 1.6}}>
                  起: {formatTime(project.startTime)}<br />
                  止: {formatTime(project.endTime)}
                </div>
              </div>
            </aside>
          </div>
        )}

        {tab === "records" && (
          <section>
            {records.length === 0 ? (
              <div className="empty-state"><p>审计日志为空</p></div>
            ) : (
              <table className="borderless-table">
                <thead>
                  <tr>
                    <th>终端标识符</th>
                    <th>提取负载内容</th>
                    <th>物理寻址 (IP)</th>
                    <th>交互时间戳</th>
                  </tr>
                </thead>
                <tbody>
                  {records.map(r => (
                    <tr key={r.id}>
                      <td><div className="td-main">{r.userId || "匿名接入"}</div><div className="td-sub">{r.fingerprint?.slice(0, 16)}</div></td>
                      <td><code className="data-group__value--code">{r.rewardContent?.slice(0, 32)}{r.rewardContent?.length > 32 ? "..." : ""}</code></td>
                      <td><div className="td-sub" style={{marginTop: 0}}>{r.ip}</div></td>
                      <td><div className="td-main">{formatTime(r.createdAt)}</div></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </section>
        )}

        {tab === "pool" && (
          <section>
            {(!project.rewardItems || project.rewardItems.length === 0) ? (
              <div className="empty-state">
                {project.rewardType === "cdk_list" ? "令牌池为空" : "当前负载类型不使用令牌池结构"}
              </div>
            ) : (
              <table className="borderless-table">
                <thead>
                  <tr>
                    <th>令牌序列片段</th>
                    <th>分配状态</th>
                    <th>目标终端</th>
                    <th>分配时间</th>
                  </tr>
                </thead>
                <tbody>
                  {project.rewardItems.map(item => (
                    <tr key={item.id} style={{opacity: item.status === "claimed" ? 0.5 : 1}}>
                      <td><code className="data-group__value--code">{item.content}</code></td>
                      <td><span className={`status-dot status--${item.status === "unused" ? "active" : "ended"}`}>{item.status === "unused" ? "待分配" : "已释放"}</span></td>
                      <td><div className="td-main">{item.claimedBy || "—"}</div></td>
                      <td><div className="td-sub" style={{marginTop: 0}}>{item.claimedAt ? formatTime(item.claimedAt) : "—"}</div></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </section>
        )}
      </div>

      <Modal
        isOpen={disableModalOpen}
        title="⚠️ 安全确认"
        variant="danger"
        confirmText="确认停用"
        cancelText="暂不操作"
        onConfirm={confirmDisable}
        onCancel={() => setDisableModalOpen(false)}
      >
        停用后该项目将立即拒绝所有领取请求。已产生的领取记录不受影响。确认停用？
      </Modal>
    </div>
  );
}
