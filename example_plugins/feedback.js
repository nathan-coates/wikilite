/**
 * 04-feedback.js
 * Adds a Like/Dislike feedback mechanism to the bottom of articles.
 */

const KEY_PREFIX_VOTE = "vote:";
const KEY_PREFIX_COUNT = "count:";

function getVoteKey(slug, email) {
    return KEY_PREFIX_VOTE + slug + ":" + email;
}

function getCountKey(slug, type) {
    return KEY_PREFIX_COUNT + slug + ":" + type;
}

function onArticleRender(html, ctx) {
    if (!ctx.Slug) return html;

    const likeCountStr = Host.storage.get(getCountKey(ctx.Slug, "like")) || "0";
    const dislikeCountStr =
        Host.storage.get(getCountKey(ctx.Slug, "dislike")) || "0";

    let userVote = "";
    if (ctx.User && ctx.User.email) {
        userVote = Host.storage.get(getVoteKey(ctx.Slug, ctx.User.email));
    }

    const likeActive = userVote === "like" ? "active" : "";
    const dislikeActive = userVote === "dislike" ? "active" : "";

    const widget = `
        <div class="plugin-feedback-container">
            <div class="plugin-feedback-label">Was this article helpful?</div>
            <div class="plugin-feedback-buttons">
                <button 
                    class="plugin-btn-feedback ${likeActive}" 
                    id="btn-like"
                    onclick="submitFeedback('${ctx.Slug}', 'like')"
                    title="I like this">
                    <span class="icon">👍</span>
                    <span class="count" id="count-like">${likeCountStr}</span>
                </button>
                <button 
                    class="plugin-btn-feedback ${dislikeActive}" 
                    id="btn-dislike"
                    onclick="submitFeedback('${ctx.Slug}', 'dislike')"
                    title="I dislike this">
                    <span class="icon">👎</span>
                    <span class="count" id="count-dislike">${dislikeCountStr}</span>
                </button>
            </div>
        </div>

        <style>
            .plugin-feedback-container {
                margin-top: 3rem;
                padding: 1.5rem 0;
                border-top: 1px solid var(--border, #eaeaea);
                display: flex;
                flex-direction: column;
                align-items: flex-start;
                font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            }
            .plugin-feedback-label {
                font-size: 0.9rem;
                font-weight: 600;
                color: #555;
                margin-bottom: 0.75rem;
            }
            .plugin-feedback-buttons {
                display: flex;
                gap: 10px;
            }
            .plugin-btn-feedback {
                background: #fff;
                border: 1px solid #ccc;
                border-radius: 6px;
                padding: 6px 12px;
                cursor: pointer;
                font-size: 0.95rem;
                display: flex;
                align-items: center;
                gap: 6px;
                transition: all 0.2s ease;
                color: #333;
            }
            .plugin-btn-feedback:hover {
                background: #f5f5f5;
                border-color: #bbb;
            }
            
            .plugin-btn-feedback.active {
                background: #e7f5ff;
                border-color: #0066cc;
                color: #0066cc;
                font-weight: 500;
            }
            
            .plugin-btn-feedback .icon {
                font-size: 1.1rem;
            }
        </style>

        <script>
            async function submitFeedback(slug, type) {
                const btnLike = document.getElementById('btn-like');
                const btnDislike = document.getElementById('btn-dislike');

                document.body.style.cursor = 'wait';

                try {
                    const resp = await fetch('/api/plugin/feedback/vote?slug=' + slug, {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ type: type })
                    });
                    
                    document.body.style.cursor = 'default';

                    if (resp.status === 401) {
                        alert("You must be logged in to vote.");
                        return;
                    }

                    const data = await resp.json();
                    
                    if (data.error) {
                        console.error(data.error);
                        return;
                    }

                    document.getElementById('count-like').innerText = data.likes;
                    document.getElementById('count-dislike').innerText = data.dislikes;

                    btnLike.classList.remove('active');
                    btnDislike.classList.remove('active');

                    if (data.userVote === 'like') {
                        btnLike.classList.add('active');
                    } else if (data.userVote === 'dislike') {
                        btnDislike.classList.add('active');
                    }

                } catch (err) {
                    console.error(err);
                    document.body.style.cursor = 'default';
                }
            }
        </script>
    `;

    return html + widget;
}

function onAction(action, payload, ctx) {
    if (action !== "vote") {
        return { error: "Unknown action" };
    }

    if (!ctx.User || !ctx.User.email) {
        return { error: "Unauthorized" };
    }

    const slug = ctx.Slug;
    const email = ctx.User.email;
    const newVoteType = payload.type;

    if (!slug) return { error: "No article slug found" };
    if (newVoteType !== "like" && newVoteType !== "dislike") {
        return { error: "Invalid vote type" };
    }

    const voteKey = getVoteKey(slug, email);
    const likeCountKey = getCountKey(slug, "like");
    const dislikeCountKey = getCountKey(slug, "dislike");

    let currentVote = Host.storage.get(voteKey);
    let likes = parseInt(Host.storage.get(likeCountKey) || "0", 10);
    let dislikes = parseInt(Host.storage.get(dislikeCountKey) || "0", 10);

    if (currentVote === newVoteType) {
        if (currentVote === "like") likes--;
        if (currentVote === "dislike") dislikes--;

        currentVote = "";
        Host.storage.delete(voteKey);
    } else {
        if (currentVote === "like") likes--;
        if (currentVote === "dislike") dislikes--;

        if (newVoteType === "like") likes++;
        if (newVoteType === "dislike") dislikes++;

        currentVote = newVoteType;
        Host.storage.set(voteKey, currentVote);
    }

    if (likes < 0) likes = 0;
    if (dislikes < 0) dislikes = 0;

    Host.storage.set(likeCountKey, likes.toString());
    Host.storage.set(dislikeCountKey, dislikes.toString());

    return {
        likes: likes,
        dislikes: dislikes,
        userVote: currentVote,
    };
}
