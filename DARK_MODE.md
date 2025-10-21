# Dark Mode Implementation

## Overview

Dark mode has been implemented in the DMR-Nexus dashboard with system preference detection. Users can choose between three modes:
- **System** (default): Automatically follows the operating system's color scheme preference
- **Light**: Always use light mode
- **Dark**: Always use dark mode

## Features

### System Preference Detection
- Detects the user's OS color scheme preference using `prefers-color-scheme` media query
- Automatically switches when the system preference changes (if "System" mode is selected)
- Preference is stored in `localStorage` and persists across sessions

### Theme Toggle
- Icon button in the header to cycle through themes: System → Light → Dark → System
- Different icons for each mode:
  - System: Monitor icon
  - Light: Sun icon
  - Dark: Moon icon

### Settings Page
- Dedicated settings page with radio buttons for theme selection
- Clear descriptions for each mode

## Implementation Details

### Tailwind Configuration
- Enabled class-based dark mode in `tailwind.config.js`
- All components use Tailwind's `dark:` variant for dark mode styles

### Store (Pinia)
- Theme state managed in `stores/app.js`
- `isDark` getter computes the effective dark mode state
- `setTheme()` action updates theme and applies it to the DOM
- `initTheme()` initializes on app load and sets up system preference listener

### Component Updates
All components have been updated with dark mode support:

#### App.vue
- Dark mode background and text colors
- Theme toggle button in header
- Smooth color transitions

#### Views
- **Dashboard**: Cards with dark backgrounds, borders, and status indicators
- **Peers**: Peer cards with state badges, metrics, and subscriptions
- **Bridges**: Bridge tables with dark mode styling
- **Activity**: Real-time activity log with event type badges
- **Settings**: Theme preference selector

#### HeaderNav
- Navigation with dark borders and hover states
- Active route highlighting

## Color Palette

### Light Mode
- Background: `gray-50`
- Cards: `white`
- Text: `gray-900`
- Borders: `gray-200`
- Accents: `blue-500`, `green-600`, etc.

### Dark Mode
- Background: `gray-900`
- Cards: `gray-800`
- Text: `gray-100`
- Borders: `gray-700`
- Accents: `blue-400`, `green-400`, etc. (slightly lighter for better contrast)

## Usage

### For Users
1. Click the theme icon in the header to cycle through modes
2. Or go to Settings page to select a specific theme
3. The preference is automatically saved

### For Developers
To add dark mode to a new component:

```vue
<template>
  <div class="bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100">
    <button class="bg-blue-500 dark:bg-blue-600 hover:bg-blue-600 dark:hover:bg-blue-700">
      Click me
    </button>
  </div>
</template>
```

### Accessing Theme State
```javascript
import { useAppStore } from '@/stores/app'

const store = useAppStore()
console.log(store.theme) // 'system' | 'light' | 'dark'
console.log(store.isDark) // boolean
store.setTheme('dark') // Change theme
```

## Browser Support

Dark mode works in all modern browsers that support:
- CSS `prefers-color-scheme` media query
- `localStorage`
- CSS custom properties (CSS variables)

Gracefully degrades to light mode in older browsers.

## Performance

- Theme application is instant (no flash of unstyled content)
- Uses CSS classes for theme switching (no JS-based style injection)
- System preference listener is registered once on app initialization
- LocalStorage prevents re-detection on every page load

## Future Enhancements

Potential improvements:
- Custom color themes beyond light/dark
- Per-component theme overrides
- Accessibility improvements (high contrast mode)
- Theme animations/transitions
