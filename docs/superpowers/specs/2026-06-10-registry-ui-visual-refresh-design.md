# Registry UI Visual Refresh Design

## Goal

Upgrade the current frontend from a set of Bootstrap-styled pages into a cohesive productized admin console. The redesign should make the site feel modern and polished while preserving fast scanability for registry browsing, tag inspection, event review, and configuration tasks.

## Scope

This design covers the server-rendered frontend templates and shared CSS for:

- Global application shell
- Catalog page
- Statistics page
- Login page
- Shared visual patterns used by the remaining pages

This effort is intentionally focused on visual hierarchy, layout consistency, and component polish. It does not change backend behavior, routing, or page-level feature scope.

## Current Problems

- The UI already uses cards, gradients, and icons, but they are applied page by page without a shared system.
- Visual emphasis is inconsistent. Many sections compete equally for attention.
- Navigation, breadcrumbs, search, tables, badges, and pagination do not feel like parts of one product.
- Several templates hardcode decorative gradients inline, which makes the look harder to maintain and less cohesive.
- Dark mode support exists, but the contrast and depth rules are not consistently tuned as a complete theme.

## Design Direction

The target direction is a productized control console:

- Modern and refined rather than playful
- Visually bold enough to feel like a real product
- Structured for dense information, not a marketing site
- Consistent across bright and dark themes

The design should favor one strong visual language over many local flourishes.

## Global Visual Language

### Brand and Tone

- Use a restrained primary accent for navigation state, links, focus state, and key actions.
- Use a small set of functional colors for success, warning, info, and danger states only.
- Replace template-specific rainbow gradients with a controlled palette and shared surface styles.
- Preserve a professional operations-tool tone even when making the UI more polished.

### Surfaces and Depth

- Introduce a layered app background so pages no longer sit on a flat Bootstrap canvas.
- Use rounded primary surfaces with soft shadows and clear borders to define content blocks.
- Keep visual depth consistent across cards, toolbars, tables, and overlays.
- Make dark mode a first-class surface system rather than a simple color inversion.

### Typography and Rhythm

- Strengthen hierarchy through spacing, weight, and scale rather than decorative color.
- Standardize heading, subheading, body, muted text, and metadata styles.
- Use a consistent vertical rhythm between hero/context areas, toolbars, content cards, and footer.

## Global Layout

### App Shell

- Replace the current heavy dark navbar with a lighter, more product-like top shell.
- Keep primary navigation in a compact horizontal layout with clearer active and hover states.
- Retain the dark-mode toggle and auth actions, but integrate them into the same visual system.
- Keep the footer minimal and aligned with the new surface/background language.

### Page Structure

Each major page should follow the same rhythm:

1. Global top navigation
2. Page context area
3. Toolbar or summary row when needed
4. Content cards
5. Footer

The page context area should absorb breadcrumbs, title context, and top-level search or status where appropriate. This creates a predictable structure across catalog, statistics, logs, and settings pages.

## Shared Components

### Toolbar Pattern

- Breadcrumbs, search, filters, and small page actions should share one toolbar pattern.
- The toolbar should feel like a product control strip rather than ad hoc Bootstrap fragments.
- Search inputs should have clearer affordance and stronger visual integration with surrounding controls.

### Card Pattern

- Replace one-off card headers and inline gradients with reusable card variants.
- Card headers should support title, supporting copy, and optional actions.
- Cards should be visually quieter so dense content remains readable.

### Table and List Pattern

- Tables are the core interaction layer and should become the most stable visual pattern in the app.
- Improve row spacing, line separation, hover treatment, and metadata styling.
- Reduce unnecessary icon and color noise while preserving fast scanning.
- Treat repository and tag sections as product lists rather than plain Bootstrap tables.

### Badges and Status

- Standardize count badges, status badges, and metadata chips.
- Use danger styling only where risk is real, such as destructive actions or failed jobs.
- Lower the default weight of destructive controls so they do not dominate list pages.

### Pagination and DataTables Controls

- Restyle pagination, length selector, and info text as part of the shared design system.
- Align control sizing, radius, spacing, and borders with the new surface language.
- Preserve DataTables behavior while making it feel native to the app.

## Page-Specific Design

### Catalog Page

The catalog page should become the main showcase for the new system.

- Introduce a clearer context band at the top containing breadcrumb navigation, page context, and shared search.
- Present repositories, tags, and recent activity as parallel module types from the same system.
- Make row items feel deliberate: primary label, secondary metadata, and tertiary actions.
- Keep delete-tag actions visible but visually subordinate until needed.
- Preserve scan speed for nested repository browsing and tag-heavy views.

### Statistics Page

- Convert the current colorful summary blocks into a unified KPI strip or grid.
- Use shared metric-card styling instead of per-card decorative gradients.
- Separate high-level summary from operational detail using module hierarchy.
- Present background jobs, purge status, and top repositories as second-level content cards under the KPI area.

### Login Page

- Give login its own focused entry composition while keeping it within the same product identity.
- Use the shared palette, form styling, button styling, and surface treatment.
- Make the form feel intentional and premium rather than a default card centered on a blank page.

### Remaining Pages

Options, event log, purge log, image detail, and related pages should not get custom visual concepts. They should inherit the same shell, toolbar, card, and table language so the whole app reads as one system.

## Motion and Interaction

- Add subtle transitions for hover, focus, theme switch feedback, and card emphasis.
- Avoid flashy animation. Motion should support clarity and perceived quality.
- Focus states must remain obvious and keyboard-friendly.

## Dark Mode

- Define dark-mode surface, border, text, hover, and muted-text values explicitly.
- Avoid using light-mode shadows and border assumptions in dark mode.
- Ensure cards, tables, toolbar controls, and footer remain legible and layered.

## Implementation Constraints

- Preserve the current server-rendered architecture.
- Favor shared CSS variables and reusable classes over repeated inline styles.
- Minimize template-specific styling rules where a system-level class can replace them.
- Do not introduce unrelated structural refactors beyond what is needed to support the redesigned frontend.

## Files Likely To Change

- `templates/base.html`
- `templates/catalog.html`
- `templates/statistics.html`
- `templates/login.html`
- `templates/options.html`
- `templates/event_log.html`
- `templates/purge_log.html`
- `templates/image_info.html`
- `templates/json_to_table.html`
- `templates/breadcrumb.html`
- `static/css/custom.css`

Additional template updates are acceptable if required to apply the shared layout consistently.

## Validation

The redesign is successful when:

- The app reads as one product rather than a collection of individually decorated pages.
- Catalog, statistics, and login clearly share the same design system.
- Table-heavy workflows remain easy to scan and use.
- Dark mode feels intentional, not incidental.
- The result is a visibly bolder UI without reducing clarity for operational use.
