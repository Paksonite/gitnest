package ui

import (
	"path/filepath"
	"strings"

	"github.com/Paksonite/gitnest/internal/gitx"
)

// Icons come in three sets (config "icons"):
//   - symbols (the default): Unicode symbols found in the fonts every system
//     ships, e.g. DejaVu Sans Mono on Debian and Ubuntu, Menlo on macOS — no
//     emoji, each one cell wide;
//   - nerd: Nerd Font glyphs for fonts such as MesloLGS NF. Only glyphs from
//     the Font Awesome, Octicons, Devicons and Seti ranges are used, which
//     are present in both Nerd Fonts v2 and v3;
//   - plain: no file, branch or folder icons at all.
const (
	iconsSymbols = iota
	iconsNerd
	iconsPlain
)

var iconSet = iconsSymbols

// iconModes are the config names of the sets, in the order i cycles them.
var iconModes = [...]string{iconsSymbols: "symbols", iconsNerd: "nerd", iconsPlain: "plain"}

func setIcons(mode string) {
	iconSet = iconsSymbols
	for i, m := range iconModes {
		if strings.EqualFold(mode, m) {
			iconSet = i
		}
	}
}

// symbols stand in for the Nerd Font glyphs in the symbols set: one shape
// per kind of file (the colour from fileIconStyled tells kinds apart too).
var symbols = map[string]string{
	nfFolder: "■", nfFolderOpen: "□", nfFile: "○",
	nfGo: "◇", nfPython: "◇", nfJS: "◇", nfTS: "◇", nfRuby: "◇", nfRust: "◇", nfJava: "◇",
	nfFileCode: "◇", nfHTML: "◇", nfCSS: "◇",
	nfJSON: "≡", nfConfig: "≡", nfCubes: "≡", nfDocker: "≡", nfGnu: "≡",
	nfMarkdown: "¶", nfFileText: "¶", nfLicense: "¶",
	nfTerminal: "$", nfGit: "±", nfDatabase: "◈",
	nfFileImage: "▧", nfFilePDF: "▤", nfFileArchive: "▦", nfLock: "⊘", nfKey: "⊗",
	nfBranchIcon: "⌥", nfRemoteBranch: "⇅", nfTag: "⚑",
}

// glyph renders a Nerd Font glyph in the current set ("" for plain).
func glyph(nf string) string {
	switch iconSet {
	case iconsNerd:
		return nf
	case iconsSymbols:
		return symbols[nf]
	}
	return ""
}

const (
	nfFolder       = ""
	nfFolderOpen   = ""
	nfFile         = ""
	nfFileText     = ""
	nfFileCode     = ""
	nfFileImage    = ""
	nfFilePDF      = ""
	nfFileArchive  = ""
	nfLock         = ""
	nfGit          = ""
	nfGo           = ""
	nfPython       = ""
	nfJS           = ""
	nfTS           = ""
	nfJSON         = ""
	nfConfig       = ""
	nfMarkdown     = ""
	nfHTML         = ""
	nfCSS          = ""
	nfTerminal     = ""
	nfDocker       = ""
	nfDatabase     = ""
	nfRuby         = ""
	nfRust         = ""
	nfJava         = ""
	nfCubes        = "" // terraform, hcl
	nfGnu          = "" // Makefile
	nfKey          = "" // certificates, keys
	nfLicense      = "" // book
	nfCommit       = ""
	nfMerge        = ""
	nfPencil       = ""
	nfRotate       = ""
	nfHistory      = ""
	nfExchange     = ""
	nfDownload     = ""
	nfPlusCircle   = ""
	nfUndo         = ""
	nfRepo         = ""
	nfDot          = ""
	nfBranchIcon   = ""
	nfRemoteBranch = "" // cloud
	nfTag          = "\uf02b"
)

var nfByExt = map[string]string{
	".go": nfGo, ".mod": nfGo, ".sum": nfGo,
	".py": nfPython, ".js": nfJS, ".mjs": nfJS, ".cjs": nfJS, ".jsx": nfJS,
	".ts": nfTS, ".tsx": nfTS,
	".json": nfJSON, ".yaml": nfConfig, ".yml": nfConfig, ".toml": nfConfig, ".ini": nfConfig,
	".cfg": nfConfig, ".conf": nfConfig, ".env": nfConfig, ".properties": nfConfig, ".xml": nfFileCode,
	".j2": nfFileCode, ".jinja": nfFileCode, ".jinja2": nfFileCode, ".tpl": nfFileCode, ".tmpl": nfFileCode,
	".md": nfMarkdown, ".markdown": nfMarkdown, ".rst": nfFileText, ".txt": nfFileText, ".log": nfFileText,
	".html": nfHTML, ".htm": nfHTML, ".css": nfCSS, ".scss": nfCSS,
	".sh": nfTerminal, ".bash": nfTerminal, ".zsh": nfTerminal, ".fish": nfTerminal, ".ps1": nfTerminal,
	".sql": nfDatabase, ".rb": nfRuby, ".rs": nfRust, ".java": nfJava, ".kt": nfJava,
	".tf": nfCubes, ".tfvars": nfCubes, ".hcl": nfCubes, ".nomad": nfCubes,
	".png": nfFileImage, ".jpg": nfFileImage, ".jpeg": nfFileImage, ".gif": nfFileImage, ".svg": nfFileImage,
	".webp": nfFileImage, ".ico": nfFileImage, ".pdf": nfFilePDF,
	".zip": nfFileArchive, ".gz": nfFileArchive, ".tgz": nfFileArchive, ".tar": nfFileArchive, ".xz": nfFileArchive,
	".lock": nfLock, ".pem": nfKey, ".crt": nfKey, ".key": nfKey, ".pub": nfKey, ".gpg": nfKey,
	".c": nfFileCode, ".h": nfFileCode, ".cpp": nfFileCode, ".cs": nfFileCode, ".swift": nfFileCode, ".lua": nfFileCode,
}

var nfByName = map[string]string{
	".gitignore": nfGit, ".gitattributes": nfGit, ".gitmodules": nfGit, ".gitkeep": nfGit,
	"dockerfile": nfDocker, "containerfile": nfDocker, "docker-compose.yml": nfDocker, "docker-compose.yaml": nfDocker,
	"makefile": nfGnu, "gnumakefile": nfGnu, "license": nfLicense, "licence": nfLicense, "copying": nfLicense,
	"go.mod": nfGo, "go.sum": nfGo, "readme": nfMarkdown,
}

// fileKind is the Nerd Font glyph for a file or directory, which names its
// kind in every icon set.
func fileKind(path string, dir, collapsed bool) string {
	if dir {
		if collapsed {
			return nfFolder
		}
		return nfFolderOpen
	}
	base := strings.ToLower(filepath.Base(path))
	if ic, ok := nfByName[base]; ok {
		return ic
	}
	if strings.HasPrefix(base, "dockerfile") {
		return nfDocker
	}
	if ic, ok := nfByExt[filepath.Ext(base)]; ok {
		return ic
	}
	return nfFile
}

// fileIcon is the icon shown before a file or directory, followed by a
// space; empty with plain icons.
func fileIcon(path string, dir, collapsed bool) string {
	if g := glyph(fileKind(path, dir, collapsed)); g != "" {
		return g + " "
	}
	return ""
}

// fileIconStyled is fileIcon in a colour for its kind of file: code, config,
// docs, shell, images, folders.
func fileIconStyled(path string, dir, collapsed bool) (plain, styled string) {
	plain = fileIcon(path, dir, collapsed)
	if plain == "" {
		return "", ""
	}
	c := cDim
	if dir {
		c = cCyan
	} else {
		switch fileKind(path, false, false) {
		case nfGo, nfPython, nfJS, nfTS, nfRuby, nfRust, nfJava, nfFileCode, nfHTML, nfCSS:
			c = cCyan
		case nfJSON, nfConfig, nfCubes, nfDocker, nfGnu:
			c = cYellow
		case nfMarkdown, nfFileText, nfLicense:
			c = cAccent
		case nfTerminal, nfGit:
			c = cGreen
		case nfFileImage, nfFilePDF, nfFileArchive, nfLock, nfKey:
			c = cRed
		}
	}
	return plain, fg(c, plain)
}

// refGlyph is the icon for a reflog action.
func refGlyph(k gitx.RefKind) string {
	if iconSet == iconsNerd {
		switch k {
		case gitx.RefCommit:
			return nfCommit
		case gitx.RefAmend:
			return nfPencil
		case gitx.RefMerge:
			return nfMerge
		case gitx.RefRebase:
			return nfRotate
		case gitx.RefReset:
			return nfHistory
		case gitx.RefCheckout:
			return nfExchange
		case gitx.RefPull:
			return nfDownload
		case gitx.RefCherryPick:
			return nfPlusCircle
		case gitx.RefRevert:
			return nfUndo
		case gitx.RefInitial:
			return nfRepo
		}
		return nfDot
	}
	switch k {
	case gitx.RefCommit:
		return "●"
	case gitx.RefAmend:
		return "✎"
	case gitx.RefMerge:
		return "◆"
	case gitx.RefRebase:
		return "↻"
	case gitx.RefReset:
		return "↶"
	case gitx.RefCheckout:
		return "→"
	case gitx.RefPull:
		return "↓"
	case gitx.RefCherryPick:
		return "✚"
	case gitx.RefRevert:
		return "↺"
	case gitx.RefInitial:
		return "◎"
	}
	return "·"
}

// branchGlyph prefixes branch names: local or remote.
func branchGlyph(remote bool) string {
	g := glyph(nfBranchIcon)
	if remote {
		g = glyph(nfRemoteBranch)
	}
	if g == "" {
		return ""
	}
	return g + " "
}

// tagGlyph prefixes tag names.
func tagGlyph() string {
	if g := glyph(nfTag); g != "" {
		return g + " "
	}
	return "⚑ "
}
