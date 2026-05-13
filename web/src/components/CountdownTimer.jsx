import { useEffect, useRef, useState } from "react";

function pad(value) {
  return String(value).padStart(2, "0");
}

function getCountdownParts(targetTime) {
  const target = new Date(targetTime).getTime();
  const diff = Number.isFinite(target) ? Math.max(0, target - Date.now()) : 0;
  const totalSeconds = Math.floor(diff / 1000);

  return {
    days: Math.floor(totalSeconds / 86400),
    hours: Math.floor((totalSeconds % 86400) / 3600),
    minutes: Math.floor((totalSeconds % 3600) / 60),
    seconds: totalSeconds % 60,
  };
}

function formatCountdown(parts) {
  const clock = `${pad(parts.hours)}:${pad(parts.minutes)}:${pad(parts.seconds)}`;
  if (parts.days > 0) {
    return `${parts.days}天 ${clock}`;
  }
  return clock;
}

function getStatusText(status, parts) {
  if (status === "upcoming") {
    return `距开始 ${formatCountdown(parts)}`;
  }
  if (status === "active") {
    return "实时发放中";
  }
  if (status === "soldout") {
    return "库存已清空";
  }
  return "活动已结束";
}

function calcFill(stats) {
  const total = Number(stats?.distributableTotal ?? stats?.maxCount ?? 0);
  const claimed = Number(stats?.claimedTotal ?? 0);
  if (!Number.isFinite(total) || total <= 0 || !Number.isFinite(claimed) || claimed <= 0) {
    return 0;
  }
  return Math.min(1, Math.max(0, claimed / total));
}

function useAnimatedNumber(value, duration = 420) {
  const safeValue = Number.isFinite(value) ? value : 0;
  const [display, setDisplay] = useState(safeValue);
  const lastValueRef = useRef(safeValue);

  useEffect(() => {
    const from = lastValueRef.current;
    const to = safeValue;
    if (from === to) {
      return undefined;
    }

    let rafId = 0;
    const start = performance.now();
    const step = (now) => {
      const progress = Math.min(1, (now - start) / duration);
      const eased = 1 - (1 - progress) * (1 - progress);
      setDisplay(Math.round(from + (to - from) * eased));
      if (progress < 1) {
        rafId = window.requestAnimationFrame(step);
      } else {
        lastValueRef.current = to;
      }
    };

    rafId = window.requestAnimationFrame(step);
    return () => window.cancelAnimationFrame(rafId);
  }, [safeValue, duration]);

  return display;
}

export default function CountdownTimer({ targetTime, status, stats, loading, className }) {
  const [parts, setParts] = useState(() => getCountdownParts(targetTime));

  useEffect(() => {
    setParts(getCountdownParts(targetTime));
    const timer = window.setInterval(() => {
      setParts(getCountdownParts(targetTime));
    }, 1000);
    return () => window.clearInterval(timer);
  }, [targetTime]);

  const total = Number(stats?.distributableTotal ?? stats?.maxCount ?? 0);
  const remaining = Number(stats?.remainingTotal ?? 0);
  const claimed = Number(stats?.claimedTotal ?? 0);
  const fill = calcFill(stats);
  const remainingAnimated = useAnimatedNumber(Math.max(0, remaining));
  const claimedAnimated = useAnimatedNumber(Math.max(0, claimed));
  const remainingLabel = total > 0 ? `${remainingAnimated} / ${total}` : "--";
  const claimedLabel = total > 0 ? `${claimedAnimated} / ${total}` : "--";

  if (loading) {
    return (
      <section className={`activity-status-line activity-status-line--loading ${className || ""}`} aria-label="状态加载中">
        <div className="activity-status-line__meta">
          <span className="skeleton skeleton--chip" />
          <span className="skeleton skeleton--chip" />
          <span className="skeleton skeleton--chip" />
        </div>
        <div className="activity-status-line__track">
          <span className="is-skeleton" />
        </div>
      </section>
    );
  }

  const statusText = getStatusText(status, parts);

  return (
    <section className={`activity-status-line ${className || ""}`} aria-live="polite">
      <div className="activity-status-line__meta">
        <span className="activity-status-line__item">{statusText}</span>
        <span className="activity-status-line__dot" />
        <span className="activity-status-line__item">
          剩余 <strong className="activity-status-line__number">{remainingLabel}</strong>
        </span>
        <span className="activity-status-line__dot" />
        <span className="activity-status-line__item">
          已领取 <strong className="activity-status-line__number">{claimedLabel}</strong>
        </span>
      </div>
      <div className="activity-status-line__track" role="progressbar" aria-valuemin={0} aria-valuemax={100} aria-valuenow={Math.round(fill * 100)}>
        <span style={{ transform: `scaleX(${fill})` }} />
      </div>
    </section>
  );
}
