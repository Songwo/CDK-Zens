import { useState, useEffect } from 'react';

/**
 * 帖子内嵌领取微组件 (Widget)
 * 显示进度条、领取按钮、资格提示
 */
export default function WelfareWidget({
  projectCode,
  name,
  totalStock = 0,
  claimedCount = 0,
  rules = '',
  requireReply = false,
  requireLike = false,
  requireFollow = false,
  minReplyLength = 0,
  userClaimed = false,
  userRewardContent = '',
  onClaim,
  onReply,
  onLike,
  onFollow,
  forumPostId,
}) {
  const [claiming, setClaiming] = useState(false);
  const [claimResult, setClaimResult] = useState(null);
  const [eligibility, setEligibility] = useState(null);
  const [copied, setCopied] = useState(false);

  const remaining = totalStock - claimedCount;
  const isSoldout = remaining <= 0;
  const progressPercent = Math.min(100, (claimedCount / totalStock) * 100);
  const canClaim = !isSoldout && !userClaimed && !claiming;

  const statusClass = isSoldout
    ? 'status--soldout'
    : userClaimed
    ? 'status--claimed'
    : 'status--active';

  const statusText = isSoldout ? '已领完' : userClaimed ? '已领取' : '进行中';

  // 获取规则列表
  const rulesList = rules
    ? rules.split('\n').filter(r => r.trim())
    : [];

  // 构建前置条件列表
  const prerequisites = [];
  if (requireReply) prerequisites.push('回复帖子');
  if (requireLike) prerequisites.push('点赞帖子');
  if (requireFollow) prerequisites.push('关注发布者');
  if (minReplyLength > 0) prerequisites.push(`回复至少 ${minReplyLength} 字`);

  const handleClaim = async () => {
    if (!canClaim) return;

    setClaiming(true);
    try {
      const result = await onClaim?.({
        projectCode,
        forumPostId,
      });
      setClaimResult(result);
    } catch (error) {
      setClaimResult({
        success: false,
        message: error.message || '领取失败',
      });
    } finally {
      setClaiming(false);
    }
  };

  const copyCDK = async () => {
    if (claimResult?.rewardContent) {
      await navigator.clipboard.writeText(claimResult.rewardContent);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const scrollToReply = () => {
    const replyBox = document.querySelector('.reply-editor');
    replyBox?.scrollIntoView({ behavior: 'smooth' });
    replyBox?.focus();
    onReply?.();
  };

  return (
    <div className={`welfare-widget ${isSoldout ? 'is-soldout' : ''}`}>
      {/* 头部 */}
      <div className="widget-header">
        <div className="widget-badge">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M20 12v6a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2v-6" />
            <polyline points="12 15 12 3" />
            <path d="M20 9H4a2 2 0 0 1 0-4h16a2 2 0 0 1 0 4z" />
          </svg>
          <span>CDK 福利</span>
        </div>
        <div className={`widget-status ${statusClass}`}>
          {statusText}
        </div>
      </div>

      {/* 进度条 */}
      <div className="widget-progress">
        <div className="progress-bar">
          <div
            className={`progress-fill ${remaining <= 3 && remaining > 0 ? 'is-low' : ''}`}
            style={{ width: `${progressPercent}%` }}
          />
        </div>
        <div className="progress-meta">
          <span className="progress-claimed">已领 {claimedCount}</span>
          <span className="progress-remaining">
            剩余 <strong>{remaining}</strong> / {totalStock}
          </span>
        </div>
      </div>

      {/* 前置条件提示 */}
      {prerequisites.length > 0 && !userClaimed && (
        <div className="widget-prerequisites">
          <div className="prerequisites-title">领取条件</div>
          <div className="prerequisites-list">
            {prerequisites.map((item, index) => (
              <div key={index} className="prerequisite-item">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <circle cx="12" cy="12" r="10" />
                  <path d="M12 6v6l4 2" />
                </svg>
                <span>{item}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* 规则说明 */}
      {rulesList.length > 0 && (
        <div className="widget-rules">
          <div className="rules-title">活动规则</div>
          {rulesList.map((rule, index) => (
            <div key={index} className="rule-item">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <polyline points="9 11 12 14 22 4" />
                <path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11" />
              </svg>
              <span>{rule}</span>
            </div>
          ))}
        </div>
      )}

      {/* 领取按钮 */}
      <button
        className="widget-claim-btn"
        disabled={!canClaim}
        onClick={handleClaim}
      >
        {claiming ? (
          <>
            <span className="btn-spinner" />
            领取中...
          </>
        ) : isSoldout ? (
          '已被领完'
        ) : userClaimed ? (
          <>
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <polyline points="20 6 9 17 4 12" />
            </svg>
            已领取
          </>
        ) : (
          '立即领取'
        )}
      </button>

      {/* 资格不足提示 */}
      {eligibility && !eligibility.eligible && (
        <div className="widget-eligibility-hint">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
            <circle cx="12" cy="12" r="10" />
            <line x1="12" y1="8" x2="12" y2="12" />
            <line x1="12" y1="16" x2="12.01" y2="16" />
          </svg>
          <span>{eligibility.reason}</span>
          {eligibility.needReply && (
            <button className="hint-action" onClick={scrollToReply}>
              去回复
            </button>
          )}
          {eligibility.needLike && (
            <button className="hint-action" onClick={onLike}>
              去点赞
            </button>
          )}
          {eligibility.needFollow && (
            <button className="hint-action" onClick={onFollow}>
              去关注
            </button>
          )}
        </div>
      )}

      {/* 领取结果 */}
      {claimResult && (
        <div className={`widget-result ${claimResult.success ? 'is-success' : 'is-error'}`}>
          {claimResult.success ? (
            <>
              <div className="result-icon">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14" />
                  <polyline points="22 4 12 14.01 9 11.01" />
                </svg>
              </div>
              <div className="result-content">
                <p className="result-title">领取成功！</p>
                <div className="result-cdk">
                  <code>{claimResult.rewardContent}</code>
                  <button className="copy-btn" onClick={copyCDK}>
                    {copied ? '已复制' : '复制'}
                  </button>
                </div>
              </div>
            </>
          ) : (
            <p>{claimResult.message}</p>
          )}
        </div>
      )}
    </div>
  );
}
