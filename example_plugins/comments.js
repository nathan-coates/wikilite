/**
 * comments.js
 * * Key Schema: "comments/{slug}/{iso_timestamp}_{user_email}"
 * Ordering: Relies on Host.storage.list()
 */

const PAGE_SIZE = 5;
const KEY_PREFIX = "comments/";

function getPrefix(slug) {
    return KEY_PREFIX + slug + "/";
}

function renderCommentHTML(key, c, isAdmin, slug) {
    const safeAuthor = DOMPurify.sanitize(c.author);
    const safeText = DOMPurify.sanitize(c.text);
    const date = dayjs(c.date).format("MMM D, YYYY h:mm A");

    let deleteBtn = "";
    if (isAdmin) {
        deleteBtn =
            `<button class="plugin-comment-delete" onclick="deleteSortedComment('${slug}', '${key}')">Delete</button>`;
    }

    return `
        <div class="plugin-comment" id="comment-${key}">
            <div class="plugin-comment-header">
                <div>
                    <strong>${safeAuthor}</strong> 
                    <span class="plugin-comment-date">${date}</span>
                </div>
                ${deleteBtn}
            </div>
            <div class="plugin-comment-body">${safeText}</div>
        </div>`;
}

function onArticleRender(html, ctx) {
    if (!ctx.Slug) return html;

    const prefix = getPrefix(ctx.Slug);
    let keys = Host.storage.list(prefix);

    keys.reverse();

    const totalComments = keys.length;
    const visibleKeys = keys.slice(0, PAGE_SIZE);
    const isAdmin = ctx.User && ctx.User.role === 3;

    let commentsHtml = "";
    for (const key of visibleKeys) {
        const jsonStr = Host.storage.get(key);
        if (jsonStr) {
            try {
                const c = JSON.parse(jsonStr);
                commentsHtml += renderCommentHTML(key, c, isAdmin, ctx.Slug);
            } catch (e) {}
        }
    }

    let loadMoreBtn = "";
    if (totalComments > PAGE_SIZE) {
        loadMoreBtn = `
            <div class="plugin-comment-loadmore">
                <button id="btn-load-more" class="btn btn-outline" onclick="loadMoreSorted('${ctx.Slug}')">Load More</button>
            </div>`;
    }

    let formHtml = "";
    if (ctx.User) {
        formHtml = `
            <form onsubmit="postSortedComment(event, '${ctx.Slug}')" class="plugin-comment-form">
                <textarea name="text" placeholder="Write a comment..." required></textarea>
                <button type="submit" class="btn">Add Comment</button>
            </form>
        `;
    } else {
        formHtml =
            `<p class="plugin-comment-login"><a href="/login">Login</a> to post comments.</p>`;
    }

    const widgetHtml = `
        <style>
            .plugin-comments-section { margin-top: 40px; padding-top: 20px; border-top: 1px solid #eee; }
            .plugin-comment { background: #f9f9f9; border: 1px solid #e0e0e0; border-radius: 4px; padding: 10px; margin-bottom: 10px; }
            .plugin-comment-header { font-size: 0.85rem; color: #555; margin-bottom: 5px; display: flex; justify-content: space-between; align-items: center; }
            .plugin-comment-date { margin-left: 8px; color: #888; }
            .plugin-comment-delete { background: none; border: none; color: #dc3545; cursor: pointer; font-size: 0.8rem; }
            .plugin-comment-delete:hover { text-decoration: underline; }
            .plugin-comment-form textarea { width: 100%; min-height: 80px; margin-bottom: 10px; padding: 8px; border: 1px solid #ccc; border-radius: 4px; font-family: inherit; }
            .plugin-comment-login { font-style: italic; color: #666; }
            .plugin-comment-loadmore { text-align: center; margin: 15px 0; }
        </style>
        <div class="plugin-comments-section">
            <h3>Comments (${totalComments})</h3>
            <div class="plugin-comments-list">${commentsHtml}</div>
            ${loadMoreBtn}
            ${formHtml}
        </div>
        <script>
        window.sortedPage = 1;

        async function loadMoreSorted(slug) {
            const btn = document.getElementById('btn-load-more');
            if(!btn) return;
            
            btn.innerText = "Loading...";
            btn.disabled = true;
            const nextPage = window.sortedPage + 1;

            try {
                const resp = await fetch('/api/plugin/comments/list?slug=' + slug, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ page: nextPage })
                });
                
                const res = await resp.json();
                
                if (res.html) {
                    const list = document.querySelector('.plugin-comments-list');
                    list.insertAdjacentHTML('beforeend', res.html);
                    window.sortedPage = nextPage;
                    
                    if (!res.hasMore) {
                        btn.parentElement.remove();
                    } else {
                        btn.innerText = "Load More";
                        btn.disabled = false;
                    }
                }
            } catch (err) { console.error(err); btn.innerText = "Error"; }
        }

        async function postSortedComment(e, slug) {
            e.preventDefault();
            const btn = e.target.querySelector('button');
            const text = e.target.text.value;
            btn.disabled = true; btn.innerText = "Posting...";

            try {
                const resp = await fetch('/api/plugin/comments/submit?slug=' + slug, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ text: text })
                });
                if (resp.ok) window.location.reload();
                else alert("Failed.");
            } catch (err) { console.error(err); alert("Error."); }
        }

        async function deleteSortedComment(slug, key) {
            if (!confirm("Delete?")) return;
            try {
                const resp = await fetch('/api/plugin/comments/delete?slug=' + slug, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ key: key })
                });
                if (resp.ok) window.location.reload();
            } catch (err) { console.error(err); }
        }
        </script>
    `;

    return html + widgetHtml;
}

function onAction(action, payload, ctx) {
    if (!ctx.Slug) return { error: "No slug" };
    const prefix = getPrefix(ctx.Slug);

    if (action === "submit") {
        if (!ctx.User) return { error: "Unauthorized" };
        if (!payload.text) return { error: "Empty" };

        const timestamp = new Date().toISOString();
        const key = prefix + timestamp + "_" + ctx.User.email;

        const data = {
            author: ctx.User.name || ctx.User.email,
            text: payload.text,
            date: timestamp,
        };

        Host.storage.set(key, JSON.stringify(data));
        return { success: true };
    }

    if (action === "list") {
        let keys = Host.storage.list(prefix);
        keys.reverse();

        const page = payload.page || 1;
        const start = (page - 1) * PAGE_SIZE;
        const end = start + PAGE_SIZE;
        const slice = keys.slice(start, end);
        const isAdmin = ctx.User && ctx.User.role === 3;

        let html = "";
        for (const key of slice) {
            const jsonStr = Host.storage.get(key);
            if (jsonStr) {
                try {
                    const c = JSON.parse(jsonStr);
                    html += renderCommentHTML(key, c, isAdmin, ctx.Slug);
                } catch (e) {}
            }
        }

        return { html: html, hasMore: end < keys.length };
    }

    if (action === "delete") {
        if (!ctx.User || ctx.User.role !== 3) return { error: "Forbidden" };
        if (!payload.key) return { error: "Missing key" };

        if (payload.key.indexOf(prefix) !== 0) {
            return { error: "Invalid key scope" };
        }

        Host.storage.delete(payload.key);
        return { success: true };
    }

    return { error: "Unknown action" };
}
