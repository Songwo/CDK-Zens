import { useEffect, useState } from "react";

function delay(ms) {
  return new Promise((resolve) => window.setTimeout(resolve, ms));
}

const LABELS = {
  upcoming: "等待开始",
  active: "立即领取",
  soldout: "已抢完",
  ended: "已结束",
};

export default function ClaimButton({ status, disabledReason, onClaim, onClaimSuccess, onClaimError }) {
  const [loading, setLoading] = useState(false);
  const [feedback, setFeedback] = useState("idle");
  const canClaim = status === "active" && !disabledReason && !loading;

  useEffect(() => {
    if (feedback === "idle") {
      return undefined;
    }

    const timer = window.setTimeout(() => {
      setFeedback("idle");
    }, 1600);
    return () => window.clearTimeout(timer);
  }, [feedback]);

  const label =
    loading
      ? "Validating..."
      : feedback === "success"
        ? "Success"
        : feedback === "wait"
          ? "Wait"
          : LABELS[status] || LABELS.upcoming;

  const hint =
    loading
      ? "正在校验活动状态并锁定激活码。"
      : feedback === "success"
        ? "领取完成，激活码已返回到终端回显区。"
        : feedback === "wait"
          ? "服务繁忙，请稍后再次点击。"
          : disabledReason || "点击后进入实时校验态。";

  async function handleClick() {
    if (!canClaim) {
      return;
    }

    setFeedback("idle");
    setLoading(true);

    try {
      // 等待一小会儿，让按钮先进入 loading 状态
      await delay(100);

      const result = await Promise.allSettled([Promise.resolve().then(onClaim), delay(400)]);
      const claimResult = result[0];

      if (claimResult.status === "fulfilled" && claimResult.value) {
        const { code, claimedAt } = claimResult.value;
        setFeedback("success");

        if (onClaimSuccess) {
          onClaimSuccess(code, claimedAt);
        }
      } else {
        setFeedback("wait");
        if (onClaimError) {
          onClaimError("服务繁忙，请稍后再次点击");
        }
      }
    } catch (error) {
      setFeedback("wait");
      if (onClaimError) {
        onClaimError(error.message || "网络请求失败，请检查网络连接");
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="claim-button-group">
      <button
        type="button"
        onClick={handleClick}
        disabled={!canClaim}
        className={`app-button app-button--primary ${!canClaim ? "is-disabled" : ""} ${canClaim ? "is-live" : ""} ${loading ? "is-validating" : ""} ${feedback !== "idle" ? `is-${feedback}` : ""}`}
      >
        <span className="app-button__scan" aria-hidden="true" />
        {label}
      </button>
      <div className="claim-progress">
        <span className={loading || canClaim ? "is-loading" : ""} />
      </div>
      <p className="claim-hint">{hint}</p>
      <p className="claim-heat">{status === "active" ? "⚡ 当前高并发通道在线" : "⚡ 通道待命中"}</p>
    </div>
  );
}
