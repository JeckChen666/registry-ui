# Registry UI Visual Refresh Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Redesign the existing server-rendered frontend into a cohesive, modern control console with a unified shell, shared component system, and polished page layouts across the main registry workflows.

**Architecture:** Keep the existing Go-rendered template architecture and DataTables behavior, but replace page-local decoration with a shared visual system expressed through reusable template structure and centralized CSS variables/classes. The work is split into shell/layout changes first, then shared components, then page-level adoption so each stage can be verified independently.

**Tech Stack:** Go HTML templates, Bootstrap 5, Bootstrap Icons, DataTables, custom CSS

---

## File Structure

- Modify: `templates/base.html`
  - Own the global shell, navigation, footer, app background, theme toggle behavior, and shared content wrapper.
- Modify: `templates/breadcrumb.html`
  - Align breadcrumb markup with the new toolbar pattern.
- Modify: `static/css/custom.css`
  - Define design tokens, light/dark surfaces, shared card/table/toolbar/form/button styles, and DataTables integration styling.
- Modify: `templates/catalog.html`
  - Apply the new context band, shared toolbar, and content-card patterns to the main repository/tag/activity page.
- Modify: `templates/statistics.html`
  - Convert summary cards and tables to the shared KPI and module styles.
- Modify: `templates/login.html`
  - Create a branded login composition using the same shell language.
- Modify: `templates/event_log.html`
  - Adopt the shared toolbar, filter bar, and table card patterns.
- Modify: `templates/options.html`
  - Apply shared page header and card/table structure.
- Modify: `templates/purge_log.html`
  - Apply shared page header, info panel, and log table styling.
- Modify: `templates/image_info.html`
  - Replace inline styles and one-off panels with shared detail-page cards.

### Task 1: Build the global shell and design token system

**Files:**
- Modify: `templates/base.html`
- Modify: `static/css/custom.css`

- [ ] **Step 1: Inspect the current shell markup before editing**

Run: `Get-Content -Raw templates\base.html`
Expected: The current file shows a dark Bootstrap navbar, simple container main section, light footer, and inline theme-toggle script.

- [ ] **Step 2: Replace the shell markup with the new app chrome**

Use this `templates/base.html` structure:

```html
<!DOCTYPE html>
<html lang="zh" data-bs-theme="light">
    <head>
        <meta charset="utf-8">
        <meta http-equiv="X-UA-Compatible" content="IE=edge">
        <meta name="viewport" content="width=device-width, initial-scale=1">
        <title>Registry UI</title>
        <link rel="stylesheet" type="text/css" href="{{ basePath }}/static/css/bootstrap.min.css">
        <link rel="stylesheet" type="text/css" href="{{ basePath }}/static/css/bootstrap-icons.min.css">
        <link rel="stylesheet" type="text/css" href="{{ basePath }}/static/css/datatables.min.css"/>
        <link rel="stylesheet" type="text/css" href="{{ basePath }}/static/css/custom.css?v={{version}}">
        <script type="text/javascript" src="{{ basePath }}/static/js/datatables.min.js"></script>
        {{yield head()}}
    </head>
    <body class="app-body">
        <div class="app-shell">
            <header class="app-header">
                <div class="container app-header-inner">
                    <a class="app-brand" href="{{ basePath }}/">
                        <span class="app-brand-mark"><i class="bi bi-box-seam"></i></span>
                        <span class="app-brand-copy">
                            <span class="app-brand-title">Registry UI</span>
                            <span class="app-brand-subtitle">Container Registry Console</span>
                        </span>
                    </a>
                    <nav class="app-nav" aria-label="Primary">
                        {{if eventsAllowed}}
                        <a class="app-nav-link" href="{{ basePath }}/__event-log">
                            <i class="bi bi-activity"></i>
                            <span>事件日志</span>
                        </a>
                        <a class="app-nav-link" href="{{ basePath }}/__purge-log">
                            <i class="bi bi-trash3"></i>
                            <span>清理日志</span>
                        </a>
                        <a class="app-nav-link" href="{{ basePath }}/__statistics">
                            <i class="bi bi-bar-chart"></i>
                            <span>统计信息</span>
                        </a>
                        <a class="app-nav-link" href="{{ basePath }}/__options">
                            <i class="bi bi-sliders"></i>
                            <span>配置选项</span>
                        </a>
                        {{end}}
                    </nav>
                    <div class="app-header-actions">
                        <button class="btn btn-app-icon" id="darkModeToggle" aria-label="切换暗色模式">
                            <i class="bi bi-moon-stars-fill" id="darkModeIcon"></i>
                        </button>
                        {{if user}}
                        <a class="btn btn-app-ghost" href="{{ basePath }}/logout" title="退出登录 ({{user}})">
                            <i class="bi bi-box-arrow-right"></i>
                            <span>退出登录</span>
                        </a>
                        {{end}}
                    </div>
                </div>
            </header>

            <main class="app-main flex-grow-1">
                <div class="container">
                    <div class="app-content">
                        {{yield body()}}
                    </div>
                </div>
            </main>

            <footer class="app-footer">
                <div class="container app-footer-inner">
                    <div class="app-footer-copy">Registry UI v{{version}}</div>
                    <div class="app-footer-links">
                        <a href="https://quiq.com" target="_blank" class="text-decoration-none">Quiq Inc.</a>
                        <a href="https://github.com/Quiq/registry-ui" target="_blank" class="text-decoration-none">
                            <i class="bi bi-github"></i>
                            <span>GitHub</span>
                        </a>
                    </div>
                </div>
            </footer>
        </div>

        <script type="text/javascript" src="{{ basePath }}/static/js/bootstrap.bundle.min.js"></script>
        <script type="text/javascript">
            (function() {
                const html = document.documentElement;
                const toggle = document.getElementById('darkModeToggle');
                const icon = document.getElementById('darkModeIcon');
                const navLinks = document.querySelectorAll('.app-nav-link');
                const currentPath = window.location.pathname.replace(/\/+$/, '') || '/';

                const savedTheme = localStorage.getItem('theme') || 'light';
                html.setAttribute('data-bs-theme', savedTheme);
                updateIcon(savedTheme);

                navLinks.forEach(function(link) {
                    const linkPath = new URL(link.href, window.location.origin).pathname.replace(/\/+$/, '') || '/';
                    if (linkPath === currentPath) {
                        link.classList.add('is-active');
                    }
                });

                toggle.addEventListener('click', function() {
                    const currentTheme = html.getAttribute('data-bs-theme');
                    const nextTheme = currentTheme === 'light' ? 'dark' : 'light';
                    html.setAttribute('data-bs-theme', nextTheme);
                    localStorage.setItem('theme', nextTheme);
                    updateIcon(nextTheme);
                });

                function updateIcon(theme) {
                    icon.className = theme === 'dark' ? 'bi-sun-fill' : 'bi-moon-stars-fill';
                }
            })();
        </script>
    </body>
</html>
```

- [ ] **Step 3: Replace `static/css/custom.css` with a shared token and shell system**

Implement a unified stylesheet that includes:

```css
:root {
    --app-bg: #f4f7fb;
    --app-bg-accent: rgba(77, 116, 255, 0.16);
    --app-surface: rgba(255, 255, 255, 0.78);
    --app-surface-strong: #ffffff;
    --app-surface-muted: #eef3fb;
    --app-border: rgba(15, 23, 42, 0.08);
    --app-border-strong: rgba(15, 23, 42, 0.12);
    --app-text: #10203a;
    --app-text-muted: #5b6b86;
    --app-heading: #0f172a;
    --app-primary: #3d63ff;
    --app-primary-strong: #2747d9;
    --app-success: #1f9d68;
    --app-info: #0ea5e9;
    --app-warning: #f59e0b;
    --app-danger: #e5484d;
    --app-shadow-sm: 0 10px 30px rgba(15, 23, 42, 0.06);
    --app-shadow-md: 0 18px 40px rgba(15, 23, 42, 0.10);
    --app-radius-lg: 24px;
    --app-radius-md: 18px;
    --app-radius-sm: 12px;
}

[data-bs-theme="dark"] {
    --app-bg: #09111f;
    --app-bg-accent: rgba(86, 129, 255, 0.22);
    --app-surface: rgba(10, 18, 33, 0.78);
    --app-surface-strong: #0d1729;
    --app-surface-muted: #101b31;
    --app-border: rgba(148, 163, 184, 0.12);
    --app-border-strong: rgba(148, 163, 184, 0.18);
    --app-text: #d9e3f3;
    --app-text-muted: #8da0bd;
    --app-heading: #f7fbff;
    --app-primary: #7c98ff;
    --app-primary-strong: #9bb0ff;
    --app-success: #33c48b;
    --app-info: #4fc3f7;
    --app-warning: #fbbf24;
    --app-danger: #ff7074;
    --app-shadow-sm: 0 14px 34px rgba(2, 6, 23, 0.34);
    --app-shadow-md: 0 22px 50px rgba(2, 6, 23, 0.42);
}

html, body {
    min-height: 100%;
}

body.app-body {
    margin: 0;
    color: var(--app-text);
    background:
        radial-gradient(circle at top left, var(--app-bg-accent), transparent 28%),
        radial-gradient(circle at top right, rgba(34, 197, 94, 0.10), transparent 22%),
        linear-gradient(180deg, var(--app-bg) 0%, color-mix(in srgb, var(--app-bg) 92%, #ffffff 8%) 100%);
    font-family: "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
}

.app-shell {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
}

.app-header {
    position: sticky;
    top: 0;
    z-index: 1000;
    backdrop-filter: blur(18px);
    background: color-mix(in srgb, var(--app-surface) 88%, transparent 12%);
    border-bottom: 1px solid var(--app-border);
}

.app-header-inner,
.app-footer-inner {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
}

.app-header-inner {
    min-height: 80px;
}

.app-brand {
    display: inline-flex;
    align-items: center;
    gap: 0.9rem;
    color: inherit;
    text-decoration: none;
}

.app-brand-mark {
    width: 2.8rem;
    height: 2.8rem;
    border-radius: 0.95rem;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    color: white;
    background: linear-gradient(135deg, var(--app-primary), #6a7cff);
    box-shadow: var(--app-shadow-sm);
}

.app-brand-copy,
.page-eyebrow,
.page-title-row,
.page-subtitle {
    display: flex;
    flex-direction: column;
}

.app-brand-title {
    font-size: 1rem;
    font-weight: 700;
    letter-spacing: 0.01em;
}

.app-brand-subtitle {
    color: var(--app-text-muted);
    font-size: 0.78rem;
}

.app-nav,
.app-header-actions,
.page-toolbar,
.page-toolbar-group,
.page-title-row,
.content-card-header,
.metric-card-head,
.list-item-main,
.empty-state {
    display: flex;
    align-items: center;
}

.app-nav {
    gap: 0.5rem;
    flex-wrap: wrap;
    justify-content: center;
}

.app-nav-link,
.btn-app-ghost,
.btn-app-icon {
    border-radius: 999px;
    border: 1px solid transparent;
    color: var(--app-text-muted);
    transition: all 0.2s ease;
}

.app-nav-link {
    display: inline-flex;
    align-items: center;
    gap: 0.55rem;
    padding: 0.7rem 0.95rem;
    text-decoration: none;
}

.app-nav-link:hover,
.app-nav-link.is-active,
.btn-app-ghost:hover,
.btn-app-icon:hover {
    color: var(--app-heading);
    background: color-mix(in srgb, var(--app-primary) 10%, var(--app-surface-strong) 90%);
    border-color: color-mix(in srgb, var(--app-primary) 18%, var(--app-border) 82%);
}

.app-nav-link.is-active {
    color: var(--app-primary-strong);
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--app-primary) 22%, transparent 78%);
}

.app-header-actions {
    gap: 0.75rem;
}

.btn-app-ghost,
.btn-app-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 0.5rem;
    background: color-mix(in srgb, var(--app-surface) 82%, transparent 18%);
    text-decoration: none;
}

.btn-app-ghost {
    padding: 0.7rem 1rem;
}

.btn-app-icon {
    width: 2.8rem;
    height: 2.8rem;
}

.app-main {
    padding: 2rem 0 2.5rem;
}

.app-content {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
}

.app-footer {
    margin-top: auto;
    padding: 1.25rem 0 2rem;
}

.app-footer-inner {
    padding: 1.1rem 1.25rem;
    border-radius: var(--app-radius-md);
    border: 1px solid var(--app-border);
    background: color-mix(in srgb, var(--app-surface) 86%, transparent 14%);
    box-shadow: var(--app-shadow-sm);
}
```

Also include classes for `.page-hero`, `.page-toolbar`, `.content-card`, `.metric-card`, `.table`, `.form-control`, `.breadcrumb`, `.status-badge`, `.count-badge`, `.empty-state`, and the DataTables selectors already used in the project.

- [ ] **Step 4: Run a quick diff to confirm the shell/token work landed cleanly**

Run: `git diff -- templates/base.html static/css/custom.css`
Expected: The diff shows the old navbar/footer styling removed and replaced with the new shell and system-level CSS classes.

- [ ] **Step 5: Commit the shell layer**

```bash
git add templates/base.html static/css/custom.css
git commit -m "feat: add unified frontend shell and design tokens"
```

### Task 2: Create shared page header and breadcrumb patterns

**Files:**
- Modify: `templates/breadcrumb.html`
- Modify: `templates/catalog.html`
- Modify: `templates/event_log.html`
- Modify: `templates/options.html`
- Modify: `templates/purge_log.html`
- Modify: `templates/statistics.html`
- Modify: `templates/image_info.html`

- [ ] **Step 1: Update breadcrumb markup to fit the new toolbar system**

Use this `templates/breadcrumb.html` markup:

```html
{{ block breadcrumb() }}
    <li class="breadcrumb-item">
        <a href="{{ basePath }}/" class="breadcrumb-link">
            <i class="bi bi-house-door"></i>
            <span>首页</span>
        </a>
    </li>
    {{if . != nil}}
        {{x := ""}}
        {{range _, p := split(., "/")}}
        {{x = x + "/" + p}}
        <li class="breadcrumb-item">
            <a href="{{ basePath }}{{ x }}" class="breadcrumb-link">{{ p }}</a>
        </li>
        {{end}}
    {{end}}
{{ end }}
```

- [ ] **Step 2: Apply a shared hero/header pattern to the catalog page**

Restructure the top of `templates/catalog.html` so the body begins with:

```html
<section class="page-hero">
    <div class="page-title-row">
        <div class="page-eyebrow">Registry Explorer</div>
        <h1 class="page-title">{{ if repoPath != "" }}{{ repoPath }}{{ else }}所有仓库{{ end }}</h1>
        <p class="page-subtitle">
            {{if repoPath != ""}}
            浏览当前仓库路径下的子仓库、标签与近期活动。
            {{else}}
            浏览镜像仓库、检查标签状态并快速定位最近活动。
            {{end}}
        </p>
    </div>
    <div class="page-toolbar">
        <nav aria-label="breadcrumb" class="page-toolbar-group page-toolbar-breadcrumb">
            <ol class="breadcrumb mb-0">
                {{ yield breadcrumb() repoPath }}
            </ol>
        </nav>
        <div class="page-toolbar-group page-toolbar-search">
            <label for="repos-search" class="visually-hidden">搜索仓库和标签</label>
            <div class="search-input-wrap">
                <i class="bi bi-search search-input-icon"></i>
                <input type="text" id="repos-search" class="form-control form-control-app" placeholder="搜索仓库、标签或活动...">
                <button type="button" class="btn btn-search-clear" id="clear-repos-search" style="display: none;">
                    <i class="bi bi-x-lg"></i>
                </button>
            </div>
        </div>
    </div>
</section>
```

- [ ] **Step 3: Apply the same hero/header pattern to the remaining major pages**

For each file, use a `page-hero` block with page-specific copy:

```html
<section class="page-hero">
    <div class="page-title-row">
        <div class="page-eyebrow">Operations</div>
        <h1 class="page-title">事件日志</h1>
        <p class="page-subtitle">筛选 Registry 事件、查看 push 记录并快速定位镜像变更。</p>
    </div>
    <div class="page-toolbar">
        <nav aria-label="breadcrumb" class="page-toolbar-group page-toolbar-breadcrumb">
            <ol class="breadcrumb mb-0">
                {{ yield breadcrumb() }}
                <li class="breadcrumb-item active" aria-current="page"><strong>事件日志</strong></li>
            </ol>
        </nav>
    </div>
</section>
```

Repeat the same structure with the right title/subtitle for:

- `templates/options.html`: `System Settings` / `配置选项`
- `templates/purge_log.html`: `Maintenance` / `清理日志`
- `templates/statistics.html`: `Insights` / `统计信息`
- `templates/image_info.html`: `Artifact Details` / image or index detail title

- [ ] **Step 4: Verify header markup consistency**

Run: `rg -n "page-hero|page-title|page-subtitle|breadcrumb-link" templates`
Expected: Each target template includes the shared `page-hero` structure and `breadcrumb-link` class.

- [ ] **Step 5: Commit the shared page-header work**

```bash
git add templates/breadcrumb.html templates/catalog.html templates/event_log.html templates/options.html templates/purge_log.html templates/statistics.html templates/image_info.html
git commit -m "feat: standardize page hero and breadcrumb layout"
```

### Task 3: Convert catalog and statistics pages to the new content-card system

**Files:**
- Modify: `templates/catalog.html`
- Modify: `templates/statistics.html`
- Modify: `static/css/custom.css`

- [ ] **Step 1: Replace catalog card blocks with shared content-card markup**

Update each catalog section to follow this pattern:

```html
<section class="content-card">
    <div class="content-card-header">
        <div>
            <div class="content-card-eyebrow">Repositories</div>
            <h2 class="content-card-title">
                <i class="bi bi-folder2"></i>
                <span>仓库</span>
            </h2>
            <p class="content-card-subtitle">当前路径下的镜像仓库与标签计数。</p>
        </div>
    </div>
    <div class="content-card-body p-0">
        <div class="table-responsive">
            <table id="datatable_repos" class="table table-app align-middle mb-0">
```

For row markup, use stronger list hierarchy:

```html
<td>
    <div class="list-item-main">
        <span class="list-item-icon"><i class="bi bi-folder2"></i></span>
        <div>
            <a href="{{ basePath }}/{{ full_repo_path }}" class="list-item-link">{{ repo }}</a>
            <div class="list-item-meta">镜像仓库</div>
        </div>
    </div>
</td>
<td><span class="count-badge">{{ tagCounts[full_repo_path] }}</span></td>
```

- [ ] **Step 2: Convert the statistics summary cards to shared metric cards**

Use this pattern at the top of `templates/statistics.html`:

```html
<div class="metrics-grid">
    <article class="metric-card">
        <div class="metric-card-head">
            <span class="metric-card-icon metric-card-icon-primary"><i class="bi bi-folder2"></i></span>
            <span class="metric-card-label">仓库</span>
        </div>
        <div class="metric-card-value">{{if repoCount > 0}}{{ repoCount }}{{else}}-{{end}}</div>
        <div class="metric-card-meta">{{if repoCount > 0}}已索引仓库总数{{else}}后台加载中...{{end}}</div>
    </article>
```

Repeat for tags and events using `metric-card-icon-success` and `metric-card-icon-info`.

- [ ] **Step 3: Restyle the operational modules under statistics**

Replace decorative headers with shared module headers:

```html
<section class="content-card">
    <div class="content-card-header">
        <div>
            <div class="content-card-eyebrow">Jobs</div>
            <h2 class="content-card-title">
                <i class="bi bi-arrow-repeat"></i>
                <span>后台任务</span>
            </h2>
            <p class="content-card-subtitle">查看目录与标签刷新任务最近运行情况。</p>
        </div>
    </div>
```

Apply the same style to purge scheduling and top-repository sections.

- [ ] **Step 4: Verify the main showcase pages use shared visual classes**

Run: `rg -n "content-card|metric-card|count-badge|list-item-link|table-app" templates/catalog.html templates/statistics.html static/css/custom.css`
Expected: The catalog and statistics pages reference only shared design classes and no inline gradient headers remain.

- [ ] **Step 5: Commit the catalog/statistics redesign**

```bash
git add templates/catalog.html templates/statistics.html static/css/custom.css
git commit -m "feat: redesign catalog and statistics surfaces"
```

### Task 4: Apply the system to login, logs, options, and detail pages

**Files:**
- Modify: `templates/login.html`
- Modify: `templates/event_log.html`
- Modify: `templates/options.html`
- Modify: `templates/purge_log.html`
- Modify: `templates/image_info.html`
- Modify: `static/css/custom.css`

- [ ] **Step 1: Convert the login page into a branded entry surface**

Use a dedicated composition like:

```html
<section class="auth-shell">
    <div class="auth-panel">
        <div class="auth-copy">
            <div class="page-eyebrow">Registry UI</div>
            <h1 class="auth-title">登录控制台</h1>
            <p class="auth-subtitle">访问镜像仓库、查看运行状态并管理清理任务。</p>
        </div>
        <div class="content-card auth-card">
            <div class="content-card-body p-4 p-lg-5">
                <h2 class="auth-card-title"><i class="bi bi-shield-lock"></i><span>身份验证</span></h2>
                {{if error}}
                <div class="alert alert-danger app-alert" role="alert">{{error}}</div>
                {{end}}
                <form method="POST" action="{{ basePath }}/login" class="auth-form">
                    <div class="mb-3">
                        <label for="username" class="form-label">用户名</label>
                        <input type="text" class="form-control form-control-app" id="username" name="username" required autofocus>
                    </div>
                    <div class="mb-4">
                        <label for="password" class="form-label">密码</label>
                        <input type="password" class="form-control form-control-app" id="password" name="password" required>
                    </div>
                    <button type="submit" class="btn btn-primary btn-app-primary w-100">登录</button>
                </form>
            </div>
        </div>
    </div>
</section>
```

- [ ] **Step 2: Convert event log and purge log into shared toolbar + filter + table cards**

For `templates/event_log.html`, wrap filters in:

```html
<section class="content-card">
    <div class="content-card-body">
        <div class="page-toolbar page-toolbar-stack">
            <div class="page-toolbar-group page-toolbar-search">
                <div class="search-input-wrap">
                    <i class="bi bi-search search-input-icon"></i>
                    <input type="text" id="events-search" class="form-control form-control-app" placeholder="搜索事件、镜像、用户或时间...">
                </div>
            </div>
            <div class="page-toolbar-group filter-switches">
```

Then render the table in a separate `content-card` using `table table-app`.

For `templates/purge_log.html`, keep the info notice but style it with `app-alert app-alert-info`, and render the log table in a shared `content-card`.

- [ ] **Step 3: Convert options and image detail pages to shared panels**

For `templates/options.html`, wrap each section in:

```html
<section class="content-card">
    <div class="content-card-header">
        <div>
            <div class="content-card-eyebrow">Configuration</div>
            <h2 class="content-card-title"><i class="bi bi-sliders"></i><span>{{ section.Name }}</span></h2>
        </div>
    </div>
```

For `templates/image_info.html`, remove the inline `<style>` block and replace it with shared detail card classes for summary, manifest, and config blocks.

- [ ] **Step 4: Verify all major templates use the unified classes**

Run: `rg -n "content-card|form-control-app|table-app|app-alert|auth-shell|search-input-wrap" templates/login.html templates/event_log.html templates/options.html templates/purge_log.html templates/image_info.html`
Expected: Each target page uses shared productized classes instead of page-local visual hacks.

- [ ] **Step 5: Commit the remaining page conversions**

```bash
git add templates/login.html templates/event_log.html templates/options.html templates/purge_log.html templates/image_info.html static/css/custom.css
git commit -m "feat: apply unified visual system across remaining pages"
```

### Task 5: Validate the redesign and clean up regressions

**Files:**
- Modify: `templates/*.html` as needed
- Modify: `static/css/custom.css` as needed

- [ ] **Step 1: Build and test the Go project**

Run: `go test ./...`
Expected: All Go tests pass without introducing backend regressions.

- [ ] **Step 2: Review the final template/style diff for leftover one-off styling**

Run: `rg -n "style=\"background: linear-gradient|<style>|bg-dark|footer.bg-light|table-light th|navbar.bg-dark" templates static/css/custom.css`
Expected: No remaining inline gradient headers or obsolete shell-specific CSS from the previous design.

- [ ] **Step 3: Check git status for only intended frontend changes**

Run: `git status --short`
Expected: Modified frontend templates, CSS, and the existing unrelated `.gitignore` change still present but untouched.

- [ ] **Step 4: Capture a final review diff**

Run: `git diff -- templates static/css/custom.css`
Expected: The diff shows a coherent migration to shared shell, card, table, and page-header patterns across the touched templates.

- [ ] **Step 5: Commit the polish pass**

```bash
git add templates static/css/custom.css
git commit -m "refactor: polish registry ui visual refresh"
```
