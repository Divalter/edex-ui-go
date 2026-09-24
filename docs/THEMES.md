# Themes

eDEX-UI-GO uses the theme format of eDEX-UI unchanged: any eDEX-UI theme works.

Themes are JSON files in the `themes` folder of the user data directory
(`~/.config/eDEX-UI-GO/themes` on Linux, `~/Library/Application Support/eDEX-UI-GO/themes` on
macOS, `%APPDATA%\eDEX-UI-GO\themes` on Windows). The 21 bundled themes are copied there at every
launch, so edit a copy under another name. Select a theme in the settings editor
(**Ctrl+Shift+S**), or click a theme file in the file browser to switch to it immediately.

## Format

```json
{
    "colors": {
        "r": 170, "g": 207, "b": 209,
        "black": "#000000",
        "light_black": "#05080d",
        "grey": "#262828",
        "red": "#ff0000",
        "yellow": "#ffff00"
    },
    "cssvars": {
        "font_main": "United Sans Medium",
        "font_main_light": "United Sans Light"
    },
    "terminal": {
        "fontFamily": "Fira Mono",
        "cursorStyle": "block",
        "foreground": "#aacfd1",
        "background": "#05080d",
        "cursor": "#aacfd1",
        "cursorAccent": "#aacfd1",
        "selection": "rgba(170,207,209,0.3)"
    },
    "globe": {
        "base": "#000000",
        "marker": "#aacfd1",
        "pin": "#aacfd1",
        "satellite": "#aacfd1"
    },
    "injectCSS": ""
}
```

| Key | Meaning |
| --- | --- |
| `colors.r/g/b` | Main UI color, used everywhere (`--color_r`, `--color_g`, `--color_b` in CSS). |
| `colors.black`, `light_black`, `grey` | Backgrounds and the grid pattern. |
| `colors.red`, `colors.yellow` | Error and warning modals (optional). |
| `colors.black` … `colors.brightWhite` | Optional ANSI palette of the terminal (`black`, `red`, `green`, `yellow`, `blue`, `magenta`, `cyan`, `white` and their `bright*` variants). Missing colors are derived from the main color. |
| `cssvars.font_main`, `font_main_light` | UI fonts. |
| `terminal.fontFamily` | Terminal font. |
| `terminal.*` | xterm.js options: `cursorStyle`, `cursorBlink`, `foreground`, `background`, `cursor`, `cursorAccent`, `selection`, `fontSize`, `fontWeight`, `fontWeightBold`, `letterSpacing`, `lineHeight`, `allowTransparency`. |
| `terminal.colorFilter` | Optional list of color operations used to derive the ANSI palette, e.g. `["negate()", "desaturate(0.4)"]` (`negate`, `grayscale`, `lighten`, `darken`, `saturate`, `desaturate`, `whiten`, `blacken`, `fade`, `opaquer`, `rotate`, `mix`). |
| `globe.*` | Colors of the network globe. |
| `injectCSS` | CSS appended to the page, to move or hide modules (see `tron-notype` or `tron-disrupted`). |

## Fonts

Fonts are loaded from the `fonts` folder of the user data directory. The file name is the font
name in lowercase, with spaces replaced by underscores, and the `.woff2` extension:
`"Fira Mono"` → `fonts/fira_mono.woff2`. Drop your own `.woff2` files there to use them.
