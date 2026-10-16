package theme

// The built-in themes. Colours follow each project's published palette; on
// the light themes a few accents are darkened so text stays readable
// (every built-in passes Check).
var builtins = []Theme{
	{Name: "catppuccin-mocha", Kind: KindDark, Palette: Palette{
		Background: "#1e1e2e", Text: "#cdd6f4", Dim: "#6c7086", Accent: "#cba6f7",
		Added: "#a6e3a1", Removed: "#f38ba8", Warn: "#f9e2af", Branch: "#89dceb",
		Sel: "#45475a", SelDim: "#313244", Border: "#6c7086",
		DiffAdd: "#a6e3a1", DiffDel: "#f38ba8", DiffHunk: "#89dceb", DiffMeta: "#cba6f7",
		Authors: []string{"#f5c2e7", "#fab387", "#94e2d5", "#89b4fa", "#b4befe", "#f2cdcd"},
	}},
	{Name: "catppuccin-latte", Kind: KindLight, Palette: Palette{
		Background: "#eff1f5", Text: "#4c4f69", Dim: "#8c8fa1", Accent: "#8839ef",
		Added: "#2f8a1f", Removed: "#d20f39", Warn: "#c35a00", Branch: "#1b7f93",
		Sel: "#ccd0da", SelDim: "#dce0e8", Border: "#8c8fa1",
		DiffAdd: "#2f8a1f", DiffDel: "#d20f39", DiffHunk: "#1b7f93", DiffMeta: "#8839ef",
		Authors: []string{"#ea76cb", "#fe640b", "#179299", "#1e66f5", "#7287fd", "#dd7878"},
	}},
	{Name: "gruvbox-dark", Kind: KindDark, Palette: Palette{
		Background: "#282828", Text: "#ebdbb2", Dim: "#928374", Accent: "#d3869b",
		Added: "#b8bb26", Removed: "#fb4934", Warn: "#fabd2f", Branch: "#83a598",
		Sel: "#504945", SelDim: "#3c3836", Border: "#928374",
		DiffAdd: "#b8bb26", DiffDel: "#fb4934", DiffHunk: "#83a598", DiffMeta: "#d3869b",
		Authors: []string{"#fe8019", "#8ec07c", "#d3869b", "#83a598", "#fabd2f", "#b8bb26"},
	}},
	{Name: "gruvbox-light", Kind: KindLight, Palette: Palette{
		Background: "#fbf1c7", Text: "#3c3836", Dim: "#7c6f64", Accent: "#8f3f71",
		Added: "#79740e", Removed: "#9d0006", Warn: "#b57614", Branch: "#076678",
		Sel: "#d5c4a1", SelDim: "#ebdbb2", Border: "#928374",
		DiffAdd: "#79740e", DiffDel: "#9d0006", DiffHunk: "#076678", DiffMeta: "#8f3f71",
		Authors: []string{"#af3a03", "#427b58", "#8f3f71", "#076678", "#b57614", "#79740e"},
	}},
	{Name: "nord", Kind: KindDark, Palette: Palette{
		Background: "#2e3440", Text: "#d8dee9", Dim: "#7b88a1", Accent: "#88c0d0",
		Added: "#a3be8c", Removed: "#bf616a", Warn: "#ebcb8b", Branch: "#81a1c1",
		Sel: "#4c566a", SelDim: "#3b4252", Border: "#616e88",
		DiffAdd: "#a3be8c", DiffDel: "#bf616a", DiffHunk: "#81a1c1", DiffMeta: "#88c0d0",
		Authors: []string{"#b48ead", "#d08770", "#8fbcbb", "#5e81ac", "#ebcb8b", "#a3be8c"},
	}},
	{Name: "solarized-dark", Kind: KindDark, Palette: Palette{
		Background: "#002b36", Text: "#93a1a1", Dim: "#657b83", Accent: "#6c71c4",
		Added: "#859900", Removed: "#dc322f", Warn: "#b58900", Branch: "#2aa198",
		Sel: "#073642", SelDim: "#01313d", Border: "#586e75",
		DiffAdd: "#859900", DiffDel: "#dc322f", DiffHunk: "#2aa198", DiffMeta: "#6c71c4",
		Authors: []string{"#d33682", "#cb4b16", "#2aa198", "#268bd2", "#b58900", "#859900"},
	}},
	{Name: "solarized-light", Kind: KindLight, Palette: Palette{
		Background: "#fdf6e3", Text: "#475b62", Dim: "#839496", Accent: "#6c71c4",
		Added: "#6c7a00", Removed: "#dc322f", Warn: "#9a7300", Branch: "#1f8a83",
		Sel: "#e4ddc8", SelDim: "#eee8d5", Border: "#93a1a1",
		DiffAdd: "#6c7a00", DiffDel: "#dc322f", DiffHunk: "#1f8a83", DiffMeta: "#6c71c4",
		Authors: []string{"#d33682", "#cb4b16", "#2aa198", "#268bd2", "#b58900", "#859900"},
	}},
	{Name: "tokyo-night", Kind: KindDark, Palette: Palette{
		Background: "#1a1b26", Text: "#c0caf5", Dim: "#565f89", Accent: "#bb9af7",
		Added: "#9ece6a", Removed: "#f7768e", Warn: "#e0af68", Branch: "#7dcfff",
		Sel: "#33467c", SelDim: "#292e42", Border: "#565f89",
		DiffAdd: "#9ece6a", DiffDel: "#f7768e", DiffHunk: "#7dcfff", DiffMeta: "#bb9af7",
		Authors: []string{"#ff9e64", "#73daca", "#7aa2f7", "#bb9af7", "#e0af68", "#9ece6a"},
	}},
	{Name: "dracula", Kind: KindDark, Palette: Palette{
		Background: "#282a36", Text: "#f8f8f2", Dim: "#6272a4", Accent: "#bd93f9",
		Added: "#50fa7b", Removed: "#ff5555", Warn: "#f1fa8c", Branch: "#8be9fd",
		Sel: "#44475a", SelDim: "#343746", Border: "#6272a4",
		DiffAdd: "#50fa7b", DiffDel: "#ff5555", DiffHunk: "#8be9fd", DiffMeta: "#bd93f9",
		Authors: []string{"#ff79c6", "#ffb86c", "#8be9fd", "#bd93f9", "#f1fa8c", "#50fa7b"},
	}},
	// terminal uses the 16 ANSI colours, so it follows whatever palette the
	// terminal is configured with (and works without true colour, e.g. in
	// Terminal.app). Empty roles keep the terminal's default colour; diffs
	// keep git's own colours.
	{Name: "terminal", Kind: KindAny, Palette: Palette{
		Text: "", Dim: "8", Accent: "5",
		Added: "2", Removed: "1", Warn: "3", Branch: "6",
		Sel: "8", SelDim: "0", Border: "8",
		Authors: []string{"13", "11", "14", "12", "10", "9"},
	}},
}
