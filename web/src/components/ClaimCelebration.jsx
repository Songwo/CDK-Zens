import { useState, useEffect, useMemo } from 'react';
import { createPortal } from 'react-dom';

/**
 * 领取成功庆祝动效组件
 * Confetti 粒子 + Checkmark 展开动画
 */
export default function ClaimCelebration({ visible, cdk, onClose }) {
  const [copied, setCopied] = useState(false);

  // 生成 Confetti 粒子配置
  const confettiParticles = useMemo(() => {
    return Array.from({ length: 30 }, (_, i) => {
      const colors = ['#f4b400', '#10b981', '#3b82f6', '#8b5cf6', '#ef4444'];
      return {
        id: i,
        color: colors[i % colors.length],
        left: `${Math.random() * 100}%`,
        delay: `${Math.random() * 0.5}s`,
        duration: `${1 + Math.random() * 1.5}s`,
        rotation: `${Math.random() * 360}deg`,
        size: 6 + Math.random() * 8,
      };
    });
  }, []);

  const copyCDK = async () => {
    if (cdk) {
      await navigator.clipboard.writeText(cdk);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const handleClose = () => {
    setCopied(false);
    onClose?.();
  };

  if (!visible) return null;

  return createPortal(
    <div className="celebration-overlay" onClick={handleClose}>
      <div className="celebration-card" onClick={(e) => e.stopPropagation()}>
        {/* Confetti 粒子 */}
        <div className="confetti-container">
          {confettiParticles.map((particle) => (
            <div
              key={particle.id}
              className="confetti"
              style={{
                '--color': particle.color,
                '--left': particle.left,
                '--delay': particle.delay,
                '--duration': particle.duration,
                '--rotation': particle.rotation,
                width: `${particle.size}px`,
                height: `${particle.size}px`,
              }}
            />
          ))}
        </div>

        {/* 成功图标 */}
        <div className="celebration-icon">
          <svg className="checkmark" viewBox="0 0 52 52">
            <circle
              className="checkmark-circle"
              cx="26"
              cy="26"
              r="25"
              fill="none"
            />
            <path
              className="checkmark-check"
              fill="none"
              d="M14.1 27.2l7.1 7.2 16.7-16.8"
            />
          </svg>
        </div>

        {/* 文案 */}
        <h2 className="celebration-title">领取成功！</h2>
        <p className="celebration-subtitle">请复制下方兑换码</p>

        {/* CDK 展示 */}
        <div className="celebration-cdk">
          <code>{cdk}</code>
          <button onClick={copyCDK}>
            {copied ? '已复制' : '复制'}
          </button>
        </div>

        <button className="celebration-close" onClick={handleClose}>
          知道了
        </button>
      </div>
    </div>,
    document.body
  );
}
