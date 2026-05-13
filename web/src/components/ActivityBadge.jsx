const STATUS_META = {
  upcoming: {
    label: "未开始",
    className: "status-badge status-badge--info",
  },
  active: {
    label: "进行中",
    className: "status-badge status-badge--success",
  },
  soldout: {
    label: "已抢完",
    className: "status-badge status-badge--warning",
  },
  ended: {
    label: "已结束",
    className: "status-badge status-badge--plain",
  },
};

export function getStatusMeta(status) {
  return STATUS_META[status] || STATUS_META.upcoming;
}

export default function ActivityBadge({ status }) {
  const meta = getStatusMeta(status);
  return <span className={meta.className}>{meta.label}</span>;
}
