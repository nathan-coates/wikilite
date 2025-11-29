/**
 * toc.js
 * Generates a Table of Contents (TOC)
 */

function onArticleRender(html, ctx) {
    const container = document.createElement("div");
    container.innerHTML = html;

    const headers = container.querySelectorAll("h1, h2, h3");
    if (headers.length === 0) return html;

    const style = document.createElement("style");
    style.textContent = `
        .plugin-toc {
            background: #f9f9f9;
            border: 1px solid #ddd;
            padding: 15px;
            margin-bottom: 20px;
            border-radius: 5px;
            font-family: system-ui, -apple-system, sans-serif;
        }
        .plugin-toc strong {
            display: block;
            margin-bottom: 10px;
            font-size: 1.1em;
        }
        .plugin-toc ul {
            list-style: none;
            padding: 0;
            margin: 0;
        }
        .plugin-toc li {
            margin-bottom: 5px;
        }
        .plugin-toc a {
            text-decoration: none;
            color: #2c3e50;
        }
        .plugin-toc a:hover {
            text-decoration: underline;
            color: #3498db;
        }
    `;

    const toc = document.createElement("div");
    toc.className = "plugin-toc";

    const title = document.createElement("strong");
    title.textContent = "Table of Contents";
    toc.appendChild(title);

    const list = document.createElement("ul");
    toc.appendChild(list);

    headers.forEach((h) => {
        if (!h.id) return;

        const li = document.createElement("li");

        let indent = 0;
        const tag = h.tagName.toUpperCase();
        if (tag === "H2") indent = 20;
        if (tag === "H3") indent = 40;

        if (indent > 0) {
            li.style.marginLeft = indent + "px";
        }

        const a = document.createElement("a");
        a.href = "#" + h.id;
        a.textContent = h.textContent;

        li.appendChild(a);
        list.appendChild(li);
    });

    toc.prepend(style);

    const article = container.querySelector("article");
    if (article) {
        article.parentNode.insertBefore(toc, article);
    } else {
        container.prepend(toc);
    }

    return container.innerHTML;
}
