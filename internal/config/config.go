package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	toml "github.com/pelletier/go-toml/v2"
)

type Config struct {
	Defaults DefaultsConfig `toml:"defaults"`
	Status   StatusConfig   `toml:"status"`
	UI       UIConfig       `toml:"ui"`
	Tmux     TmuxConfig     `toml:"tmux"`
	Keys     KeysConfig     `toml:"keys"`
	Theme    ThemeConfig    `toml:"theme"`
}

type DefaultsConfig struct {
	Agent  string `toml:"agent"`
	Layout string `toml:"layout"`
}

type StatusConfig struct {
	PollIntervalMS int `toml:"poll_interval_ms"`
	CaptureLines   int `toml:"capture_lines"`
}

type UIConfig struct {
	PreviewRefreshMS int `toml:"preview_refresh_ms"`
	SidebarMinWidth  int `toml:"sidebar_min_width"`
	SidebarMaxWidth  int `toml:"sidebar_max_width"`
}

type TmuxConfig struct {
	SessionPrefix string `toml:"session_prefix"`
	ReturnKey     string `toml:"return_key"`
}

type KeysConfig struct {
	Up        string `toml:"up"`
	Down      string `toml:"down"`
	Enter     string `toml:"enter"`
	New       string `toml:"new"`
	Delete    string `toml:"delete"`
	PaneLeft  string `toml:"pane_left"`
	PaneRight string `toml:"pane_right"`
	Broadcast string `toml:"broadcast"`
	Diff      string `toml:"diff"`
	Rename    string `toml:"rename"`
	Filter    string `toml:"filter"`
	Tab       string `toml:"tab"`
	MarkRead  string `toml:"mark_read"`
	JumpNext  string `toml:"jump_next"`
	Restart   string `toml:"restart"`
	Stop      string `toml:"stop"`
	Help      string `toml:"help"`
	Quit      string `toml:"quit"`
}

type ThemeConfig struct {
	Border     string `toml:"border"`
	AppTitle   string `toml:"app_title"`
	SelectedFG string `toml:"selected_fg"`
	SelectedBG string `toml:"selected_bg"`
	Normal     string `toml:"normal"`
	Working    string `toml:"working"`
	Waiting    string `toml:"waiting"`
	Idle       string `toml:"idle"`
	Stopped    string `toml:"stopped"`
	Error      string `toml:"error"`
	Branch     string `toml:"branch"`
	AgentName  string `toml:"agent_name"`
	HelpKey    string `toml:"help_key"`
	HelpDesc   string `toml:"help_desc"`
	Dim        string `toml:"dim"`
}

func Default() Config {
	return Config{
		Defaults: DefaultsConfig{
			Agent:  "claude",
			Layout: "agent-shell",
		},
		Status: StatusConfig{
			PollIntervalMS: 1500,
			CaptureLines:   50,
		},
		UI: UIConfig{
			PreviewRefreshMS: 750,
			SidebarMinWidth:  30,
			SidebarMaxWidth:  50,
		},
		Tmux: TmuxConfig{
			SessionPrefix: "colo-",
			ReturnKey:     "e",
		},
		Keys: KeysConfig{
			Up:        "k",
			Down:      "j",
			Enter:     "enter",
			New:       "n",
			Delete:    "d",
			PaneLeft:  "h",
			PaneRight: "l",
			Broadcast: "b",
			Diff:      "D",
			Rename:    "r",
			Filter:    "/",
			Tab:       "tab",
			MarkRead:  "m",
			JumpNext:  "J",
			Restart:   "R",
			Stop:      "s",
			Help:      "?",
			Quit:      "q",
		},
		Theme: ThemeConfig{
			Border:     "62",
			AppTitle:   "99",
			SelectedFG: "99",
			SelectedBG: "236",
			Normal:     "252",
			Working:    "82",
			Waiting:    "220",
			Idle:       "245",
			Stopped:    "240",
			Error:      "196",
			Branch:     "109",
			AgentName:  "140",
			HelpKey:    "99",
			HelpDesc:   "245",
			Dim:        "240",
		},
	}
}

func DefaultPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	return filepath.Join(dir, "colosseum", "config.toml")
}

func Load(path string) (Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("reading config %s: %w", path, err)
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing config %s: %w", path, err)
	}

	return cfg, nil
}
