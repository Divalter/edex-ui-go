import { GetTheme } from '../../../wailsjs/go/main/App.js';

export interface Theme {
    primary: string;
    background: string;
    text: string;
    error: string;
    fontMain: string;
}

export function createThemeStore() {
    let theme = $state<Theme>({
        primary: '#00ffe6',
        background: '#000000',
        text: '#00ffe6',
        error: '#ff003c',
        fontMain: 'Share Tech Mono, monospace'
    });

    return {
        get theme() {
            return theme;
        },
        async load(themeName: string = 'tron') {
            try {
                const savedTheme = await GetTheme(themeName);
                if (savedTheme && savedTheme.colors) {
                    const r = savedTheme.colors.r ?? 0;
                    const g = savedTheme.colors.g ?? 255;
                    const b = savedTheme.colors.b ?? 230;
                    theme = {
                        primary: `rgb(${r}, ${g}, ${b})`,
                        background: savedTheme.colors.black || '#000000',
                        text: `rgb(${r}, ${g}, ${b})`,
                        error: savedTheme.colors.red || '#ff003c',
                        fontMain: savedTheme.fontMain ? `${savedTheme.fontMain}, monospace` : 'Share Tech Mono, monospace'
                    };
                }
            } catch (err) {
                console.warn("Wails GetTheme not available, using default tron theme", err);
            }
        },
        apply() {
            const root = document.documentElement;
            root.style.setProperty('--color-primary', theme.primary);
            root.style.setProperty('--color-bg', theme.background);
            root.style.setProperty('--color-text', theme.text);
            root.style.setProperty('--color-border', theme.primary);
            root.style.setProperty('--color-error', theme.error);
            root.style.setProperty('--font-main', theme.fontMain);
        }
    }
}

export const themeStore = createThemeStore();
