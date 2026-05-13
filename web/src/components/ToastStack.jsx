export default function ToastStack({ toasts, onDismiss }) {
  if (!toasts.length) {
    return null;
  }

  return (
    <div className="toast-stack">
      {toasts.map((toast) => (
        <div key={toast.id} className={`toast toast--${toast.tone || "info"}`}>
          <div>
            <strong>{toast.title}</strong>
            <p>{toast.message}</p>
          </div>
          <button type="button" onClick={() => onDismiss(toast.id)}>
            关闭
          </button>
        </div>
      ))}
    </div>
  );
}
