export type AccentTheme = 'lavender' | 'violet' | 'indigo' | 'rose' | 'emerald' | 'amber' | 'cyan';
export type Density = 'comfortable' | 'compact';

export interface ThemePreferences {
  accent: AccentTheme;
  density: Density;
}

export const ACCENT_PALETTES: Record<
  AccentTheme,
  { label: string; primary: string; hover: string; subtle: string; border: string; glow: string }
> = {
  lavender: {
    label: 'Lavender',
    primary: '#a78bfa',
    hover: '#8b5cf6',
    subtle: 'rgba(167, 139, 250, 0.12)',
    border: 'rgba(167, 139, 250, 0.3)',
    glow: 'rgba(167, 139, 250, 0.25)',
  },
  violet: {
    label: 'Violet',
    primary: '#c084fc',
    hover: '#a855f7',
    subtle: 'rgba(192, 132, 252, 0.12)',
    border: 'rgba(192, 132, 252, 0.3)',
    glow: 'rgba(192, 132, 252, 0.25)',
  },
  indigo: {
    label: 'Indigo',
    primary: '#818cf8',
    hover: '#6366f1',
    subtle: 'rgba(129, 140, 248, 0.12)',
    border: 'rgba(129, 140, 248, 0.3)',
    glow: 'rgba(129, 140, 248, 0.25)',
  },
  rose: {
    label: 'Rose',
    primary: '#fb7185',
    hover: '#f43f5e',
    subtle: 'rgba(251, 113, 133, 0.12)',
    border: 'rgba(251, 113, 133, 0.3)',
    glow: 'rgba(251, 113, 133, 0.25)',
  },
  emerald: {
    label: 'Emerald',
    primary: '#34d399',
    hover: '#10b981',
    subtle: 'rgba(52, 211, 153, 0.12)',
    border: 'rgba(52, 211, 153, 0.3)',
    glow: 'rgba(52, 211, 153, 0.25)',
  },
  amber: {
    label: 'Amber',
    primary: '#fbbf24',
    hover: '#f59e0b',
    subtle: 'rgba(251, 191, 36, 0.12)',
    border: 'rgba(251, 191, 36, 0.3)',
    glow: 'rgba(251, 191, 36, 0.25)',
  },
  cyan: {
    label: 'Cyan',
    primary: '#22d3ee',
    hover: '#06b6d4',
    subtle: 'rgba(34, 211, 238, 0.12)',
    border: 'rgba(34, 211, 238, 0.3)',
    glow: 'rgba(34, 211, 238, 0.25)',
  },
};

export function loadThemePreferences(): ThemePreferences {
  if (typeof window === 'undefined') {
    return { accent: 'lavender', density: 'comfortable' };
  }
  try {
    const raw = localStorage.getItem('gofyr_theme_prefs');
    if (raw) {
      const parsed = JSON.parse(raw);
      if (parsed.accent && ACCENT_PALETTES[parsed.accent as AccentTheme]) {
        return {
          accent: parsed.accent,
          density: parsed.density || 'comfortable',
        };
      }
    }
  } catch {
    // fallback
  }
  return { accent: 'lavender', density: 'comfortable' };
}

export function saveThemePreferences(prefs: ThemePreferences): void {
  if (typeof window === 'undefined') return;
  localStorage.setItem('gofyr_theme_prefs', JSON.stringify(prefs));
  applyTheme(prefs);
}

export function applyTheme(prefs: ThemePreferences): void {
  if (typeof document === 'undefined') return;
  const palette = ACCENT_PALETTES[prefs.accent] || ACCENT_PALETTES.lavender;
  const root = document.documentElement;

  root.style.setProperty('--accent-primary', palette.primary);
  root.style.setProperty('--accent-hover', palette.hover);
  root.style.setProperty('--accent-subtle', palette.subtle);
  root.style.setProperty('--accent-border', palette.border);
  root.style.setProperty('--accent-glow', palette.glow);

  root.setAttribute('data-accent', prefs.accent);
  root.setAttribute('data-density', prefs.density);
}
