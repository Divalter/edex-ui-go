# 🎨 Themes Guide

eDEX-UI-Go features a powerful theming engine compatible with the original eDEX-UI theme format.

## 📂 Where to place themes

Themes are stored as `.json` files in your user configuration directory:
- **Windows**: `%APPDATA%\edex-ui-go\themes\`
- **macOS**: `~/Library/Application Support/edex-ui-go/themes/`
- **Linux**: `~/.config/edex-ui-go/themes/`

## 📝 Theme JSON Format

```json
{
  "name": "Tron Legacy",
  "author": "User",
  "colors": {
    "background": "#000000",
    "primary": "#00d0ff",
    "text": "#ffffff",
    "text_alt": "#a0a0a0",
    "terminal_bg": "#051515",
    "terminal_fg": "#00ffff"
  },
  "fonts": {
    "main": "Rajdhani",
    "terminal": "Fira Code"
  },
  "css": ".globe { filter: hue-rotate(180deg); }"
}
```

### Color Properties

- `background`: Main application background color.
- `primary`: Accent color used for borders, highlights, and primary text.
- `text`: Default text color.
- `text_alt`: Muted/secondary text color.
- `terminal_bg`: Background color of the terminal emulators.
- `terminal_fg`: Foreground text color of the terminal emulators.
*(You can also specify ansi colors `black`, `red`, `green`, `yellow`, `blue`, `magenta`, `cyan`, `white` and their `bright_` variants).*

### Font Configuration

You can specify system fonts or web fonts. Ensure the font is available on your system or placed in the `fonts` directory within your config folder.

### Custom CSS Injection

The `css` property allows you to inject raw CSS to override any specific styles in the application. Use this carefully!
