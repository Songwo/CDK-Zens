import { useState, useRef, useCallback } from 'react';

/**
 * CDK 批量粘贴输入框组件
 * 支持多行 CDK 粘贴，自动识别、去重、预览
 */
export default function CDKBulkTextarea({ value = [], onChange, placeholder }) {
  const [inputText, setInputText] = useState('');
  const textareaRef = useRef(null);
  const maxPreview = 8;

  const defaultPlaceholder = `粘贴兑换码，每行一个...
例如：
ABCD-EFGH-IJKL
MNOP-QRST-UVWX
1234-5678-9012`;

  const parseCDKs = useCallback((text) => {
    const lines = text
      .split(/[\n\r]+/)
      .map(line => line.trim())
      .filter(line => line.length > 0);

    // 去重
    return [...new Set(lines)];
  }, []);

  const handleTextChange = (e) => {
    const newText = e.target.value;
    setInputText(newText);
    const parsed = parseCDKs(newText);
    onChange?.(parsed);
  };

  const handlePaste = (e) => {
    // 粘贴后自动解析
    setTimeout(() => {
      const textarea = textareaRef.current;
      if (textarea) {
        const parsed = parseCDKs(textarea.value);
        onChange?.(parsed);
      }
    }, 0);
  };

  const removeCDK = (index) => {
    const newList = [...value];
    newList.splice(index, 1);
    setInputText(newList.join('\n'));
    onChange?.(newList);
  };

  const clearAll = () => {
    setInputText('');
    onChange?.([]);
  };

  const maskCDK = (cdk) => {
    if (cdk.length <= 8) return cdk;
    return cdk.substring(0, 4) + '****' + cdk.substring(cdk.length - 4);
  };

  const previewCDKs = value.slice(0, maxPreview);

  return (
    <div className="cdk-textarea-wrap">
      <div className="cdk-textarea-header">
        <span className="cdk-count">
          {value.length > 0 ? (
            <>
              已识别 <strong>{value.length}</strong> 个兑换码
            </>
          ) : (
            '粘贴兑换码，每行一个'
          )}
        </span>
        {value.length > 0 && (
          <button className="cdk-clear-btn" onClick={clearAll}>
            清空
          </button>
        )}
      </div>

      <textarea
        ref={textareaRef}
        value={inputText}
        onChange={handleTextChange}
        onPaste={handlePaste}
        className="cdk-textarea"
        placeholder={placeholder || defaultPlaceholder}
        rows={6}
      />

      {value.length > 0 && (
        <div className="cdk-preview">
          {previewCDKs.map((cdk, index) => (
            <div key={index} className="cdk-chip">
              <span className="cdk-chip__text">{maskCDK(cdk)}</span>
              <button
                className="cdk-chip__remove"
                onClick={() => removeCDK(index)}
              >
                ×
              </button>
            </div>
          ))}
          {value.length > maxPreview && (
            <div className="cdk-more">+{value.length - maxPreview} 个</div>
          )}
        </div>
      )}
    </div>
  );
}
