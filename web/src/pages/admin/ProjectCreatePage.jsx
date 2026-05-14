import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { adminApi } from "../../lib/api";

function formatDateTimeLocal(date) {
  const p = (v) => String(v).padStart(2, "0");
  return `${date.getFullYear()}-${p(date.getMonth() + 1)}-${p(date.getDate())}T${p(date.getHours())}:${p(date.getMinutes())}`;
}

const INITIAL_FORM = {
  name: "", description: "", startTime: "", endTime: "",
  slug: "",
  totalStock: 100, perUserLimit: 1, rewardType: "cdk_list",
  rewardContent: "", rewardListText: "",
  enabled: true, needLogin: false, needBindIdentity: false, rules: "",
  requireCaptcha: true, showStock: true, showEndTime: true,
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

export default function ProjectCreatePage() {
  const navigate = useNavigate();
  const [form, setForm] = useState(INITIAL_FORM);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [createdUrl, setCreatedUrl] = useState("");
  const [toast, setToast] = useState("");

  function patch(key, value) {
    setForm(prev => ({ ...prev, [key]: value }));
  }

  function slugify(value) {
    return String(value || "airdrop")
      .trim()
      .toLowerCase()
      .replace(/[^a-z0-9\u4e00-\u9fa5]+/g, "-")
      .replace(/^-+|-+$/g, "")
      .replace(/[\u4e00-\u9fa5]/g, "")
      || `airdrop-${Date.now().toString(36)}`;
  }

  function applyQuickTime(type) {
    const start = new Date();
    const end = new Date();
    if (type === "7d") end.setDate(end.getDate() + 7);
    else if (type === "30d") end.setDate(end.getDate() + 30);
    else if (type === "forever") end.setFullYear(end.getFullYear() + 20);
    else end.setHours(23, 59, 0, 0);
    patch("startTime", formatDateTimeLocal(start));
    patch("endTime", formatDateTimeLocal(end));
  }

  async function handleSubmit(e) {
    e.preventDefault();
    setError("");

    if (!form.name.trim()) { setError("节点标识 (项目名称) 不能为空"); return; }
    if (!form.startTime) { setError("必须设定激活时间边界"); return; }

    setSaving(true);
    try {
      const rewardList = form.rewardType === "cdk_list"
        ? form.rewardListText.split(/[\r\n,;，；]+/).map(s => s.trim()).filter(Boolean)
        : [];

      const project = await adminApi.createProject({
        name: form.name.trim(),
        description: form.description,
      });
      const campaign = await adminApi.createCampaign({
        projectId: project.id,
        name: form.name.trim(),
        description: form.description,
        startAt: new Date(form.startTime).toISOString(),
        endAt: form.endTime ? new Date(form.endTime).toISOString() : "",
        enabled: form.enabled,
        allowRepeat: false,
        perUserLimit: Number(form.perUserLimit) || 1,
        perIPLimit: 0,
        perDeviceLimit: 1,
        requireCaptchaDefault: form.requireCaptcha,
        rewardList: form.rewardType === "cdk_list" ? rewardList : Array.from({ length: Number(form.totalStock) || 1 }, (_, i) => `${form.rewardContent || "MIUBOX-REWARD"}-${String(i + 1).padStart(4, "0")}`),
        rules: form.rules,
      });
      const slug = slugify(form.slug || form.name);
      const node = await adminApi.createNode({
        projectId: project.id,
        campaignId: campaign.id,
        name: form.name.trim(),
        slug,
        title: form.name.trim(),
        description: form.description,
        buttonText: "立即领取",
        showStock: form.showStock,
        showEndTime: form.showEndTime,
        requireCaptcha: form.requireCaptcha,
        status: form.enabled ? "active" : "paused",
        deviceLimitEnabled: true,
      });
      setCreatedUrl(window.location.origin + (node.claimUrl || `/claim/${slug}`));
    } catch (err) {
      setError(err.message);
    } finally {
      setSaving(false);
    }
  }

  if (createdUrl) {
    return (
      <div className="admin-page anim-fade-up">
        {toast && <Toast message={toast} onClose={() => setToast("")} />}
        <div className="success-panel">
          <h2>部署指令已下达</h2>
          <p>节点初始化成功，路由网关已为其分配专属入口。您可以将此链接分发至受众网络。</p>
          <div className="link-box">
            <input type="text" readOnly value={createdUrl} onClick={e => e.target.select()} />
            <button onClick={() => { navigator.clipboard.writeText(createdUrl); setToast("✓ 链路已复制"); }}>Copy</button>
          </div>
          <div className="btn-group" style={{justifyContent: 'center'}}>
            <button className="btn btn--secondary" onClick={() => { setCreatedUrl(""); setForm(INITIAL_FORM); }}>继续部署</button>
            <button className="btn btn--primary" onClick={() => navigate("/admin/projects")}>返回控制台</button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="admin-page anim-fade-up stagger-1">
      <header className="page-hero" style={{marginBottom: 0}}>
        <button className="btn btn--text" style={{marginLeft: '-12px', marginBottom: '16px'}} onClick={() => navigate("/admin/projects")}>← 返回列表</button>
        <h1 className="page-hero__title">部署新分发节点</h1>
        <p className="page-hero__desc">根据业务需求配置空投参数。请确保资源池设定无误，节点激活后即刻生效。</p>
      </header>

      {error && <div style={{background: '#fef2f2', color: '#dc2626', padding: '16px', borderRadius: '8px', marginTop: '24px'}}>{error}</div>}

      <form className="split-form anim-fade-up stagger-2" onSubmit={handleSubmit}>
        
        <div className="form-section">
          <div className="form-section__header">
            <h3>基础元信息</h3>
            <p>定义该节点在网路中的标识及向外部展示的公开说明文本。</p>
          </div>
          <div className="form-grid">
            <div className="field-group">
              <label>节点标识 <b>*</b></label>
              <input className="styled-input" value={form.name} onChange={e => patch("name", e.target.value)} placeholder="内部名称或公开活动标题" />
            </div>
            <div className="field-group">
              <label>领取链接 Slug</label>
              <input className="styled-input" value={form.slug} onChange={e => patch("slug", e.target.value)} placeholder="例如 freshman-2026，留空自动生成" />
            </div>
            <div className="field-group">
              <label>公开广播描述</label>
              <textarea className="styled-textarea" value={form.description} onChange={e => patch("description", e.target.value)} placeholder="此信息将在领取引导页展示给终端用户" />
            </div>
          </div>
        </div>

        <div className="form-section">
          <div className="form-section__header">
            <h3>时间边界约束</h3>
            <p>精确控制资源的释放窗口，超过生命周期的请求将被网关拒绝。</p>
          </div>
          <div className="form-grid">
            <div className="form-row-2">
              <div className="field-group">
                <label>生命周期起点 <b>*</b></label>
                <input type="datetime-local" className="styled-input" value={form.startTime} onChange={e => patch("startTime", e.target.value)} />
              </div>
              <div className="field-group">
                <label>生命周期终点</label>
                <input type="datetime-local" className="styled-input" value={form.endTime} onChange={e => patch("endTime", e.target.value)} />
              </div>
            </div>
            <div className="quick-time-bar">
              <span className="field-hint" style={{lineHeight: '28px', marginRight: '8px'}}>快速设定:</span>
              {[["today", "今日内"], ["7d", "一周"], ["30d", "月度"], ["forever", "永久驻留"]].map(([k, l]) => (
                <button type="button" key={k} className="btn btn--text" style={{padding: '4px 8px'}} onClick={() => applyQuickTime(k)}>{l}</button>
              ))}
            </div>
          </div>
        </div>

        <div className="form-section">
          <div className="form-section__header">
            <h3>资产配置策略</h3>
            <p>设定向终端用户投放的负载类型及容量限制阈值。</p>
          </div>
          <div className="form-grid">
            <div className="form-row-2">
              <div className="field-group">
                <label>负载类型</label>
                <select className="styled-select" value={form.rewardType} onChange={e => patch("rewardType", e.target.value)}>
                  <option value="cdk_list">动态令牌 (CDK池模式)</option>
                  <option value="fixed_text">静态指令 (所有人相同内容)</option>
                  <option value="link">外部跃迁 (跳转链接)</option>
                  <option value="passphrase">验证口令</option>
                </select>
              </div>
              <div className="field-group">
                <label>单源节点限制</label>
                <input type="number" className="styled-input" min={1} value={form.perUserLimit} onChange={e => patch("perUserLimit", e.target.value)} />
                <span className="field-hint">每个终端最大交互次数</span>
              </div>
            </div>

            {form.rewardType === "cdk_list" ? (
              <div className="field-group">
                <label>注入令牌序列 (CDK List)</label>
                <textarea className="styled-textarea" rows={6} value={form.rewardListText} onChange={e => patch("rewardListText", e.target.value)} placeholder="每行输入一个独立令牌序列..." />
                <span className="field-hint">当前已录入序列数量: {form.rewardListText.split(/[\r\n,;，；]+/).filter(s => s.trim()).length}</span>
              </div>
            ) : (
              <div className="form-row-2">
                <div className="field-group">
                  <label>静态负载内容</label>
                  <input type="text" className="styled-input" value={form.rewardContent} onChange={e => patch("rewardContent", e.target.value)} placeholder="填写固定文本或链接地址" />
                </div>
                <div className="field-group">
                  <label>预留总容量</label>
                  <input type="number" className="styled-input" min={1} value={form.totalStock} onChange={e => patch("totalStock", e.target.value)} />
                </div>
              </div>
            )}
            
            <div className="field-group">
              <label>终端指引条款</label>
              <textarea className="styled-textarea" value={form.rules} onChange={e => patch("rules", e.target.value)} placeholder="声明获取及使用要求" />
            </div>
          </div>
        </div>

        <div className="form-section">
          <div className="form-section__header">
            <h3>安全与访问协议</h3>
            <p>建立防腐层控制，管理访问来源的合法性。</p>
          </div>
          <div className="form-grid" style={{gap: '16px'}}>
            <label className="switch-label">
              <input type="checkbox" checked={form.enabled} onChange={e => patch("enabled", e.target.checked)} />
              <span>部署后立即连接网关 (开启分发)</span>
            </label>
            <label className="switch-label">
              <input type="checkbox" checked={form.needLogin} onChange={e => patch("needLogin", e.target.checked)} />
              <span>实施身份断言 (必须登录社区账号)</span>
            </label>
            <label className="switch-label">
              <input type="checkbox" checked={form.needBindIdentity} onChange={e => patch("needBindIdentity", e.target.checked)} />
              <span>高密校验 (强制终端设备特征绑定)</span>
            </label>
            <label className="switch-label">
              <input type="checkbox" checked={form.requireCaptcha} onChange={e => patch("requireCaptcha", e.target.checked)} />
              <span>开启 hCaptcha 人机验证</span>
            </label>
            <label className="switch-label">
              <input type="checkbox" checked={form.showStock} onChange={e => patch("showStock", e.target.checked)} />
              <span>领取页展示库存</span>
            </label>
            <label className="switch-label">
              <input type="checkbox" checked={form.showEndTime} onChange={e => patch("showEndTime", e.target.checked)} />
              <span>领取页展示结束时间</span>
            </label>
          </div>
        </div>

        <div className="form-actions border-top">
          <button type="button" className="btn btn--secondary" onClick={() => navigate("/admin/projects")}>终止部署</button>
          <button type="submit" className="btn btn--primary" disabled={saving}>{saving ? "正在向网关同步..." : "执行节点部署"}</button>
        </div>

      </form>
    </div>
  );
}
