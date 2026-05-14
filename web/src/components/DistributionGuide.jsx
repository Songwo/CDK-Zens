import { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { adminApi } from "../lib/api";
import { CopyButton, Modal, Toast } from "./ui";

const STEPS = [
  { key: "project", index: 1, title: "创建项目", desc: "先创建一个项目，用来归类活动和分发节点。", btn: "去创建项目" },
  { key: "campaign", index: 2, title: "创建活动", desc: "配置活动名称、时间范围、领取规则。", btn: "创建活动" },
  { key: "cdk", index: 3, title: "导入 CDK", desc: "把兑换码批量导入到对应活动库存中。", btn: "导入 CDK" },
  { key: "node", index: 4, title: "创建分发节点", desc: "为活动生成一个公开领取入口。", btn: "创建节点" },
  { key: "security", index: 5, title: "开启验证码 / 风控", desc: "根据需要开启 hCaptcha、IP 限制和设备限制。", btn: "配置安全策略" },
  { key: "test", index: 6, title: "复制链接并测试", desc: "复制节点链接，打开公开领取页进行测试。", btn: "查看分发链接" },
];

const STORAGE_KEY = "miubox_guide_collapsed";
const AUTO_COLLAPSE_KEY = "miubox_guide_auto_collapsed_done";
const REFRESH_EVENT = "miubox:onboarding-refresh";

function completionFromStatus(status) {
  return {
    project: !!status?.hasProject,
    campaign: !!status?.hasCampaign,
    cdk: !!status?.hasCdkStock,
    node: !!status?.hasDistributionNode,
    security: !!status?.hasRiskConfig,
    test: !!status?.hasPublicLink,
  };
}

function routeForStep(step, status) {
  const returnTo = encodeURIComponent("/admin/dashboard");
  if (step.key === "project") return `/admin/projects?open=create&returnTo=${returnTo}`;
  if (step.key === "campaign") return status?.recommendedProjectId ? `/admin/campaigns?open=create&projectId=${encodeURIComponent(status.recommendedProjectId)}&returnTo=${returnTo}` : `/admin/campaigns?open=create&returnTo=${returnTo}`;
  if (step.key === "cdk") return status?.recommendedCampaignId ? `/admin/cdks/import?campaignId=${encodeURIComponent(status.recommendedCampaignId)}&returnTo=${returnTo}` : `/admin/cdks/import?returnTo=${returnTo}`;
  if (step.key === "node") return status?.latestStockCampaignId ? `/admin/nodes/create?campaignId=${encodeURIComponent(status.latestStockCampaignId)}&returnTo=${returnTo}` : `/admin/nodes/create?returnTo=${returnTo}`;
  if (step.key === "security") return "/admin/captcha";
  return "/admin/nodes/links";
}

export function notifyOnboardingRefresh() {
  window.dispatchEvent(new Event(REFRESH_EVENT));
}

export function DistributionGuideCard() {
  const navigate = useNavigate();
  const [status, setStatus] = useState(null);
  const [loading, setLoading] = useState(true);
  const [toast, setToast] = useState(null);
  const [modalOpen, setModalOpen] = useState(false);
  const [collapsed, setCollapsed] = useState(() => {
    try { return localStorage.getItem(STORAGE_KEY) === "1"; } catch { return false; }
  });
  const [autoCollapsedDone, setAutoCollapsedDone] = useState(() => {
    try {
      return localStorage.getItem(AUTO_COLLAPSE_KEY) === "1" || localStorage.getItem(STORAGE_KEY) === "1";
    } catch {
      return false;
    }
  });

  async function reload() {
    setLoading(true);
    try {
      setStatus(await adminApi.onboardingStatus());
    } catch (err) {
      setToast({ type: "error", message: err.message || "获取向导状态失败" });
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    reload();
    window.addEventListener(REFRESH_EVENT, reload);
    window.addEventListener("focus", reload);
    return () => {
      window.removeEventListener(REFRESH_EVENT, reload);
      window.removeEventListener("focus", reload);
    };
  }, []);

  const completion = useMemo(() => completionFromStatus(status), [status]);
  const doneCount = Object.values(completion).filter(Boolean).length;
  const currentStep = status?.currentStep || 1;
  const allDone = currentStep === 7;
  const currentStepTitle = STEPS.find((step) => step.index === currentStep)?.title;

  useEffect(() => {
    if (allDone && !collapsed && !autoCollapsedDone) {
      try { localStorage.setItem(STORAGE_KEY, "1"); } catch {}
      try { localStorage.setItem(AUTO_COLLAPSE_KEY, "1"); } catch {}
      setAutoCollapsedDone(true);
      setCollapsed(true);
    }
  }, [allDone, collapsed, autoCollapsedDone]);

  function toggleCollapse() {
    const next = !collapsed;
    setCollapsed(next);
    if (allDone) {
      setAutoCollapsedDone(true);
      try { localStorage.setItem(AUTO_COLLAPSE_KEY, "1"); } catch {}
    }
    try { localStorage.setItem(STORAGE_KEY, next ? "1" : "0"); } catch {}
  }

  function handleStepClick(step) {
    if (step.key === "campaign" && !status?.hasProject) {
      setToast({ type: "error", message: "请先创建项目" });
      return;
    }
    if (step.key === "cdk" && !status?.hasCampaign) {
      setToast({ type: "error", message: "请先创建活动" });
      return;
    }
    if (step.key === "node" && !status?.hasCdkStock) {
      setToast({ type: "error", message: "请先导入 CDK" });
      return;
    }
    if (step.key === "test" && !status?.hasDistributionNode) {
      setToast({ type: "error", message: "请先创建分发节点" });
      return;
    }
    navigate(routeForStep(step, status));
  }

  return (
    <>
      {toast && <Toast {...toast} onClose={() => setToast(null)} />}
      <section className="panel distribution-guide">
        <div className="distribution-guide__header">
          <div className="distribution-guide__title-row" onClick={toggleCollapse} style={{ cursor: "pointer" }}>
            <span className={`distribution-guide__chevron ${collapsed ? "distribution-guide__chevron--collapsed" : ""}`}>⌄</span>
            <div>
              <h2>快速创建一次 CDK 分发</h2>
              {collapsed && (
                <p style={{ margin: 0 }}>
                  {loading ? "正在同步流程状态..." : allDone ? "分发流程已完成" : `进度 ${doneCount}/${STEPS.length}，建议下一步：${currentStepTitle}`}
                </p>
              )}
            </div>
          </div>
          <div className="distribution-guide__actions">
            <span className="distribution-guide__progress">{doneCount}/{STEPS.length}</span>
            <button className="btn btn--secondary" onClick={() => setModalOpen(true)}>分发向导</button>
            <button className="btn btn--text" onClick={reload}>{loading ? "同步中" : "刷新"}</button>
            <button className="btn btn--text" onClick={toggleCollapse} title={collapsed ? "展开" : "收起"}>{collapsed ? "展开" : "收起"}</button>
          </div>
        </div>

        {!collapsed && (
          <>
            <p className={`distribution-guide__subtitle ${allDone ? "distribution-guide__subtitle--done" : ""}`}>
              {allDone ? "分发流程已完成，你可以继续创建新的活动或复制已有节点链接进行投放。" : (status?.nextActionHint || "按照下面步骤完成项目、活动、库存、节点和安全配置。")}
            </p>
            <div className="distribution-guide__steps">
              {STEPS.map((step) => {
                const done = completion[step.key];
                const isCurrent = !done && step.index === currentStep;
                return (
                  <div key={step.key} className={`guide-step ${done ? "guide-step--done" : ""} ${isCurrent ? "guide-step--current" : ""}`}>
                    <div className="guide-step__icon">{done ? "✓" : step.index}</div>
                    <strong className="guide-step__title">{step.title}</strong>
                    <span className="guide-step__desc">{step.desc}</span>
                    <span className="field-hint">{done ? "已完成" : isCurrent ? "当前推荐步骤" : "未完成"}</span>
                    <button className={`btn ${isCurrent ? "btn--primary" : "btn--text"} guide-step__btn`} onClick={() => handleStepClick(step)}>
                      {done ? "查看" : step.btn}
                    </button>
                  </div>
                );
              })}
            </div>
            {status?.recommendedPublicLink && (
              <div className="distribution-guide__hint">
                最新分发链接：<CopyButton value={window.location.origin + status.recommendedPublicLink}>复制链接</CopyButton>
              </div>
            )}
          </>
        )}
      </section>

      <Modal open={modalOpen} title="分发向导 — 完整流程" onCancel={() => setModalOpen(false)}>
        <div className="guide-modal-body">
          <ol className="guide-modal-steps">
            {STEPS.map((step) => {
              const done = completion[step.key];
              const isCurrent = !done && step.index === currentStep;
              return (
                <li key={step.key} className={`guide-modal-step ${done ? "guide-modal-step--done" : ""} ${isCurrent ? "guide-modal-step--current" : ""}`}>
                  <div className="guide-modal-step__num">{done ? "✓" : step.index}</div>
                  <div className="guide-modal-step__content"><strong>{step.title}</strong><p>{step.desc}</p></div>
                  <button className={`btn ${isCurrent ? "btn--primary" : "btn--secondary"} guide-modal-step__btn`} onClick={() => { setModalOpen(false); handleStepClick(step); }}>
                    {done ? "查看" : step.btn}
                  </button>
                </li>
              );
            })}
          </ol>
        </div>
        <div className="modal-footer"><button className="modal-btn modal-btn--cancel" onClick={() => setModalOpen(false)}>关闭</button></div>
      </Modal>
    </>
  );
}
