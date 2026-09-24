import { GetConfig } from '../../../wailsjs/go/main/App.js';

export interface AppConfig {
    shell: string;
    theme: string;
    noIntro: boolean;
    termFontSize: number;
    allowWindowed: boolean;
}

export function createConfigStore() {
    let config = $state<AppConfig>({
        shell: 'bash',
        theme: 'tron',
        noIntro: false,
        termFontSize: 14,
        allowWindowed: false
    });

    return {
        get config() {
            return config;
        },
        async load() {
            try {
                const savedConfig = await GetConfig();
                if (savedConfig) {
                    config = {
                        shell: savedConfig.shell || 'bash',
                        theme: savedConfig.theme || 'tron',
                        noIntro: !!savedConfig.noIntro,
                        termFontSize: savedConfig.termFontSize || 14,
                        allowWindowed: !!savedConfig.allowWindowed
                    };
                }
            } catch (err) {
                console.warn("Wails GetConfig not available, using defaults", err);
            }
        }
    }
}

export const configStore = createConfigStore();
