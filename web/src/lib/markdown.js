function escapeHtml(input) {
  return input
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function renderInline(input) {
  return input
    .replace(/`([^`]+)`/g, "<code>$1</code>")
    .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
    .replace(/\*([^*]+)\*/g, "<em>$1</em>")
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank" rel="noreferrer">$1</a>');
}

export function renderMarkdown(input = "") {
  const lines = escapeHtml(input).split("\n");
  const result = [];
  let inList = false;

  for (const rawLine of lines) {
    const line = rawLine.trimEnd();
    const trimmed = line.trim();

    if (!trimmed) {
      if (inList) {
        result.push("</ul>");
        inList = false;
      }
      continue;
    }

    if (trimmed.startsWith("- ")) {
      if (!inList) {
        result.push("<ul>");
        inList = true;
      }
      result.push(`<li>${renderInline(trimmed.slice(2))}</li>`);
      continue;
    }

    if (inList) {
      result.push("</ul>");
      inList = false;
    }

    if (trimmed.startsWith("### ")) {
      result.push(`<h3>${renderInline(trimmed.slice(4))}</h3>`);
      continue;
    }
    if (trimmed.startsWith("## ")) {
      result.push(`<h2>${renderInline(trimmed.slice(3))}</h2>`);
      continue;
    }
    if (trimmed.startsWith("# ")) {
      result.push(`<h1>${renderInline(trimmed.slice(2))}</h1>`);
      continue;
    }
    if (trimmed.startsWith("> ")) {
      result.push(`<blockquote>${renderInline(trimmed.slice(2))}</blockquote>`);
      continue;
    }

    result.push(`<p>${renderInline(trimmed)}</p>`);
  }

  if (inList) {
    result.push("</ul>");
  }

  return result.join("");
}
