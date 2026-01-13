# Design System - Job Hunter OS

## Color Palette

### Background Colors
```
--pico-background-color: #1a1f2e  // Main background (lighter than GitHub's #0d1117)
--pico-card-background-color: #242c3d  // Cards/panels
--pico-card-separator-color: #334155  // Borders/dividers
```

### Text Colors
```
--pico-color: #e2e8f0           // Primary text (high contrast)
--pico-muted-color: #94a3b8     // Secondary text (labels, descriptions)
```

### Accent Colors
```
--pico-primary: #60a5fa          // Blue - primary actions, links
--pico-success: #22c55e          // Green - success states
--pico-warning: #f59e0b          // Orange/Yellow - warnings
--pico-error: #ef4444            // Red - errors, rejected
```

### Status Badge Colors
| Status | Background | Border | Text |
|--------|-----------|--------|------|
| RAW | `rgba(245, 158, 11, 0.2)` | `rgba(245, 158, 11, 0.3)` | `#f59e0b` |
| ANALYZED | `rgba(96, 165, 250, 0.2)` | `rgba(96, 165, 250, 0.3)` | `#60a5fa` |
| INTERESTED | `rgba(34, 197, 94, 0.2)` | `rgba(34, 197, 94, 0.3)` | `#22c55e` |
| REJECTED | `rgba(239, 68, 68, 0.2)` | `rgba(239, 68, 68, 0.3)` | `#ef4444` |
| PAUSED | `rgba(148, 163, 184, 0.2)` | `rgba(148, 163, 184, 0.3)` | `#94a3b8` |

---

## Typography

### Font Sizes
```
--pico-font-size: 16px         // Base font size
--pico-line-height: 1.6        // Line height for body text
```

### Heading Scale
| Level | Size | Weight | Usage |
|-------|------|--------|-------|
| h1 | 1.875rem (30px) | 700 | Page titles |
| h2 | 1.5rem (24px) | 600 | Section headers |
| h3 | 1.25rem (20px) | 600 | Card titles |
| small | 0.875rem (14px) | 400 | Secondary text |

### Font Weights
```
400 - Regular (body text)
500 - Medium (sidebar links)
600 - Semi-bold (h2, labels)
700 - Bold (h1, brand)
```

---

## Spacing Scale

### Margin/Padding Units
```
0.25rem = 4px   // Tight spacing
0.5rem  = 8px   // Status badges padding
0.625rem = 10px  // Sidebar link padding
1rem    = 16px  // Base unit
1.5rem  = 24px  // Card padding, section gaps
2rem    = 32px  // Article spacing
```

### Layout Spacing
| Element | Spacing |
|---------|---------|
| Card padding | `1.5rem` |
| Sidebar padding | `1rem` |
| Main content padding | `1.5rem` |
| Article bottom margin | `2rem` |
| Grid gap | `1rem` (default), `0.5rem` (tight), `1.5rem` (loose) |

---

## Border Radius

```
--pico-border-radius: 0.5rem (8px)  // Default for cards, buttons
0.25rem (4px)                           // Status badges
0.375rem (6px)                          // Buttons (legacy)
```

---

## Layout Structure

### Page Layout
```
┌─────────────────────────────────────────────┐
│  Sidebar (16rem)  │  Main Content (flex-1) │
│  - Logo/H1        │  - 1.5rem padding    │
│  - Nav Links      │  - Cards/Tables      │
│  - Footer (v0.1.0)│                      │
└─────────────────────────────────────────────┘
```

### Sidebar
- Width: `16rem` (256px)
- Background: `--pico-card-background-color`
- Links: `0.625rem` padding, `0.25rem` gap between items
- Active state: `rgba(96, 165, 250, 0.15)` background

### Responsive Breakpoints
```
Mobile: < 768px  - Sidebar becomes horizontal nav
Tablet: ≥ 768px  - 2 columns for grids
Desktop: ≥ 1024px - 4 columns for grids
```

---

## Component Standards

### Cards
```css
.card {
  background: var(--pico-card-background-color);
  border: 1px solid var(--pico-card-separator-color);
  border-radius: var(--pico-border-radius);
  padding: 1.5rem;
}
```

### Buttons
- Primary actions: Blue background
- Secondary actions: Outline style
- Danger actions: Red background
- Padding: `0.5rem 1rem`
- Border radius: `var(--pico-border-radius)`

### Status Badges
```css
.status-badge {
  padding: 0.25rem 0.625rem;
  border-radius: 0.25rem;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.025em;
}
```

### Tables
- Striped rows: `rgba(255, 255, 255, 0.02)` for even rows
- Hover effect: `rgba(255, 255, 255, 0.03)`
- Cell padding: `0.75rem 1rem`
- Border bottom between rows

### Form Elements
- Background: `rgba(255, 255, 255, 0.03)`
- Border: `1px solid var(--pico-card-separator-color)`
- Focus ring: `0 0 0 3px rgba(96, 165, 250, 0.2)`
- Placeholder: `var(--pico-muted-color)`

---

## Utility Classes

### Flexbox
```
.flex           { display: flex }
.flex-col       { flex-direction: column }
.flex-1         { flex: 1 }
.items-center   { align-items: center }
.justify-between { justify-content: space-between }
```

### Spacing
```
.gap-2 { gap: 0.5rem }
.gap-4 { gap: 1rem }
.gap-6 { gap: 1.5rem }
.mb-4  { margin-bottom: 1rem }
.mb-6  { margin-bottom: 1.5rem }
.mb-8  { margin-bottom: 2rem }
```

### Sizes
```
.h-screen { height: 100vh }
.h-full   { height: 100% }
.w-1\/3   { width: 33.333% }
```

---

## Design Principles

1. **Contrast First**: All text must meet WCAG AA contrast standards (4.5:1 for normal text)
2. **Spacing Consistency**: Use the spacing scale, don't use arbitrary values
3. **Visual Hierarchy**: Use font weight, size, and color to establish hierarchy
4. **Responsive Design**: Mobile-first approach, enhance for larger screens
5. **Dark Mode Only**: Optimize for dark theme, no light mode support needed

---

## Icon Usage

- Use SVG icons inline or via icon font
- Icon size should match adjacent text (or 16px for standalone)
- Use same color as text for consistency

---

## Animation & Transitions

```css
transition: background-color 0.15s ease, color 0.15s ease;
```

- Keep animations under 200ms for UI elements
- No animations for layout shifts
- Hover states only on interactive elements
