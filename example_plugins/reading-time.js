/**
 * reading-time.js
 * * Calculates estimated reading time and injects a badge.
 */

function onArticleRender(html, ctx) {
    const container = document.createElement("div");
    container.innerHTML = html;

    const article = container.querySelector("article") || container;
    const text = article.textContent || "";

    const words = text.trim().split(/\s+/).length;
    const minutes = Math.ceil(words / 200);

    const badge = document.createElement("div");
    badge.className = "plugin-read-time";
    badge.innerHTML = `<span>⏱️ ${minutes} min read</span>`;

    // Add embedded styles
    const style = document.createElement("style");
    style.textContent = `
        .plugin-read-time {
            font-size: 0.9em;
            color: #666;
            margin-bottom: 15px;
            font-family: sans-serif;
            display: inline-block;
            background: #eee;
            padding: 4px 8px;
            border-radius: 4px;
        }
    `;
    badge.appendChild(style);

    if (article !== container) {
        article.prepend(badge);
    } else {
        container.prepend(badge);
    }

    return container.innerHTML;
}
