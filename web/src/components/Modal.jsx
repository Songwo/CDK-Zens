import React, { useEffect } from 'react';
import { createPortal } from 'react-dom';

/**
 * Premium Modal Component
 * @param {boolean} isOpen
 * @param {string} title
 * @param {React.ReactNode} children
 * @param {string} confirmText
 * @param {string} cancelText
 * @param {string} variant - 'primary' | 'danger'
 * @param {Function} onConfirm
 * @param {Function} onCancel
 */
export default function Modal({
  isOpen,
  title,
  children,
  confirmText = "确认",
  cancelText = "取消",
  variant = "primary",
  onConfirm,
  onCancel
}) {
  useEffect(() => {
    if (!isOpen) return;
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    const onKey = (e) => {
      if (e.key === "Escape") onCancel?.();
    };
    window.addEventListener("keydown", onKey);
    return () => {
      document.body.style.overflow = prev;
      window.removeEventListener("keydown", onKey);
    };
  }, [isOpen, onCancel]);

  if (!isOpen) return null;
  const isDanger = variant === "danger";

  return createPortal(
    <div className="modal-overlay">
      <div className="modal-backdrop" onClick={() => { if (!isDanger) onCancel?.(); }} />
      <div className="modal-container" role="dialog" aria-modal="true">
        <div className="modal-header">
          <h2 className="modal-title">{title}</h2>
        </div>
        <div className="modal-body">
          {children}
        </div>
        <div className="modal-footer">
          <button className="modal-btn modal-btn--cancel" onClick={onCancel}>
            {cancelText}
          </button>
          <button
            className={`modal-btn modal-btn--${variant}`}
            onClick={onConfirm}
          >
            {confirmText}
          </button>
        </div>
      </div>
    </div>,
    document.body
  );
}
