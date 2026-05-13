import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { Modal, Toast } from "./ui";

const STEPS = [
  {
    key: "project",
    title: "创建项目",
    desc: "先创建一个项目，用来归类活动和分发节点。",
    btn: "去创建项目",
    path: "/admin/projects",
  },
  {
    key: "campaign",
    title: "创建活动",
    desc: "配置活动名称、时间范围、领取规则。",
    btn: "创建活动",
    path: "/admin/campaigns",
  },
  {
    key: "cdk",
    title: "导入 CDK",
    desc: "把兑换码批量导入到对应活动库存中。",
    btn: "导入 CDK",
    path: "/admin/cdks",
    requireCampaign: true,
  },
  {
    key: "node",
    title: "创建分发节点",
    desc: "为活动生成一个公开领取入口。",
    btn: "创建节点",
    path: "/admin/nodes",
    requireCampaign: true,
  },
  {
    key: "security",
    title: "开启验证码 / 风控",
    desc: "根据需要开启 hCaptcha、IP 限制和设备限制。",
    btn: "配置安全策略",
    path: "/admin/captcha",
  },
  {
    key: "test",
    title: "复制链接并测试",
    desc: "复制节点链接，打开公开领取页进行测试。",
    btn: "查看分发链接",
    path: "/admin/nodes",
    requireNode: true,
  },
];

function getCompletion(stats, nodes, captchaConfigured) {
  const hasActiveNode = (nodes || []).some((n) => n.status === "active");
  return {
    project: (stats.totalCampaigns || 0) > 0,
    campaign: (stats.totalCampaigns || 0) > 0,
    cdk: (stats.totalStock || 0) > 0,
    node: (stats.totalNodes || 0) > 0,
    security: captchaConfigured || (nodes || []).some((n) => n.requireCaptcha),
    test: hasActiveNode,
  };
}

function getNextStep(completion) {
  for (const step of STEPS) {
    if (!completion[step.key]) return step.key;
  }
  return null;
}

const STORAGE_KEY = "miubox_guide_collapsed";

export function DistributionGuideCard({ stats, nodes, captchaConfigured }) {
  const navigate = useNavigate();
  const [toast, setToast] = useState(null);
  const [modalOpen, setModalOpen] = useState(false);
  const [collapsed, setCollapsed] = useState(() => {
    try { return localStorage.getItem(STORAGE_KEY) === "1"; } catch { return false; }
  });

  const completion = getCompletion(stats || {}, nodes, captchaConfigured);
  const nextStepKey = getNextStep(completion);
  const allDone = !nextStepKey;
  const doneCount = Object.values(completion).filter(Boolean).length;

  // 全部完成时自动收起（仅首次）
  useEffect(() => {
    if (allDone && !collapsed) {
      try { localStorage.setItem(STORAGE_KEY, "1"); } catch {}
      setCollapsed(true);
    }
  }, [allDone]);

  function toggleCollapse() {
    const next = !collapsed;
    setCollapsed(next);
    try { localStorage.setItem(STORAGE_KEY, next ? "1" : "0"); } catch {}
  }

  function handleStepClick(step) {
    if (step.requireCampaign && !(stats?.totalCampaigns > 0)) {
      setToast({ type: "error", message: "请先创建活动，再进行此操作" });
      return;
    }
    if (step.requireNode && !(stats?.totalNodes > 0)) {
      setToast({ type: "error", message: "请先创建分发节点" });
      return;
    }
    navigate(step.path);
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
                  {allDone ? "✓ 分发流程已完成" : `进度 ${doneCount}/${STEPS.length}，建议下一步：${STEPS.find((s) => s.key === nextStepKey)?.title}`}
                </p>
              )}
            </div>
          </div>
          <div className="distribution-guide__actions">
            <span className="distribution-guide__progress">{doneCount}/{STEPS.length}</span>
            <button className="btn btn--secondary" onClick={() => setModalOpen(true)}>
              分发向导
            </button>
            <button className="btn btn--text" onClick={toggleCollapse} title={collapsed ? "展开" : "收起"}>
              {collapsed ? "展开" : "收起"}
            </button>
          </div>
        </div>

        {!collapsed && (
          <>
            {!allDone && (
              <p className="distribution-guide__subtitle">
                按照下面步骤完成项目、活动、库存、节点和安全配置，新手也能快速上线领取页。
              </p>
            )}
            {allDone && (
              <p className="distribution-guide__subtitle distribution-guide__subtitle--done">
                ✓ 分发流程已完成，你可以继续创建新的活动或复制已有节点链接进行投放。
              </p>
            )}

            <div className="distribution-guide__steps">
              {STEPS.map((step, idx) => {
                const done = completion[step.key];
                const isCurrent = step.key === nextStepKey;
                return (
                  <div
                    key={step.key}
                    className={`guide-step ${done ? "guide-step--done" : ""} ${isCurrent ? "guide-step--current" : ""}`}
                  >
                    <div className="guide-step__icon">
                      {done ? "✓" : idx + 1}
                    </div>
                    <strong className="guide-step__title">{step.title}</strong>
                    <span className="guide-step__desc">{step.desc}</span>
                    <button
                      className={`btn ${isCurrent ? "btn--primary" : "btn--text"} guide-step__btn`}
                      onClick={() => handleStepClick(step)}
                    >
                      {done ? "已完成" : step.btn}
                    </button>
                  </div>
                );
              })}
            </div>

            {nextStepKey && (
              <div className="distribution-guide__hint">
                建议下一步：<strong>{STEPS.find((s) => s.key === nextStepKey)?.title}</strong>
              </div>
            )}
          </>
        )}
      </section>

      {/* 详细分发向导 Modal */}
      <Modal open={modalOpen} title="分发向导 — 完整流程" onCancel={() => setModalOpen(false)}>
        <div className="guide-modal-body">
          {allDone && (
            <div className="guide-modal-done">
              <span className="guide-modal-done__icon">🎉</span>
              <p>分发流程已完成，你可以继续创建新的活动或复制已有节点链接进行投放。</p>
            </div>
          )}
          <ol className="guide-modal-steps">
            {STEPS.map((step, idx) => {
              const done = completion[step.key];
              const isCurrent = step.key === nextStepKey;
              return (
                <li key={step.key} className={`guide-modal-step ${done ? "guide-modal-step--done" : ""} ${isCurrent ? "guide-modal-step--current" : ""}`}>
                  <div className="guide-modal-step__num">
                    {done ? "✓" : idx + 1}
                  </div>
                  <div className="guide-modal-step__content">
                    <strong>{step.title}</strong>
                    <p>{step.desc}</p>
                  </div>
                  <button
                    className={`btn ${isCurrent ? "btn--primary" : "btn--secondary"} guide-modal-step__btn`}
                    onClick={() => { setModalOpen(false); handleStepClick(step); }}
                  >
                    {done ? "查看" : step.btn}
                  </button>
                </li>
              );
            })}
          </ol>
        </div>
        <div className="modal-footer">
          <button className="modal-btn modal-btn--cancel" onClick={() => setModalOpen(false)}>关闭</button>
        </div>
      </Modal>
    </>
  );
}
