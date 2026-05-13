function formatTime(value) {
  if (!value) {
    return "--";
  }

  return new Date(value).toLocaleString("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export default function RecordWall({ records = [], loading = false }) {
  if (loading) {
    return (
      <div className="record-list">
        {Array.from({ length: 4 }, (_, index) => (
          <div key={index} className="record-row record-row--skeleton">
            <span className="skeleton" />
            <span className="skeleton" />
            <span className="skeleton" />
          </div>
        ))}
      </div>
    );
  }

  if (!records.length) {
    return <p className="empty-line">暂无公开领取记录，第一条成功领取后会显示在这里。</p>;
  }

  return (
    <div className="record-list">
      {records.map((record) => (
        <div key={`${record.maskedCode}-${record.time}`} className="record-row">
          <span className="record-row__code">{record.maskedCode}</span>
          <span className="record-row__user">{record.displayName || "匿名用户"}</span>
          <span className="record-row__time">{formatTime(record.time)}</span>
        </div>
      ))}
    </div>
  );
}
