package theme

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed themes/*.json
var themesFS embed.FS

type TerminalTheme struct {
	FontFamily  string `json:"fontFamily"`
	CursorBlink bool   `json:"cursorBlink"`
}

type ColorsTheme struct {
	R int `json:"r"`
	G int `json:"g"`
	B int `json:"b"`
	Black string `json:"black"`
	White string `json:"white"`
	Red string `json:"red"`
	Green string `json:"green"`
	Blue string `json:"blue"`
	Cyan string `json:"cyan"`
	Magenta string `json:"magenta"`
	Yellow string `json:"yellow"`
}

type Theme struct {
	FontMain      string        `json:"fontMain"`
	FontMainLight string        `json:"fontMainLight"`
	Terminal      TerminalTheme `json:"terminal"`
	Colors        ColorsTheme   `json:"colors"`
	InjectCSS     string        `json:"injectCSS"`
}

func LoadTheme(name string) (*Theme, error) {
	data, err := themesFS.ReadFile(fmt.Sprintf("themes/%s.json", name))
	if err != nil {
		return nil, err
	}

	var theme Theme
	if err := json.Unmarshal(data, &theme); err != nil {
		return nil, err
	}

	return &theme, nil
}

func ListThemes() []string {
	entries, err := themesFS.ReadDir("themes")
	if err != nil {
		return []string{}
	}

	var themes []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			themes = append(themes, strings.TrimSuffix(entry.Name(), ".json"))
		}
	}
	return themes
}
