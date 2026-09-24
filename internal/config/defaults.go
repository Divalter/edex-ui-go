package config

import "runtime"

func DefaultConfig() *Config {
	shell := "/bin/bash"
	if runtime.GOOS == "windows" {
		shell = "powershell.exe"
	}

	return &Config{
		Shell:                shell,
		ShellArgs:            []string{},
		Cwd:                  "",
		Env:                  make(map[string]string),
		Theme:                "tron",
		Keyboard:             "en-US",
		TermFontSize:         14,
		Audio:                true,
		AudioVolume:          1.0,
		DisableFeedbackAudio: false,
		PingAddr:             "8.8.8.8",
		NoIntro:              false,
		NoCursor:             false,
		AllowWindowed:        false,
	}
}
