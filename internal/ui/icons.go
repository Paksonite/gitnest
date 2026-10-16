package ui

import (
	"path/filepath"
	"strings"

	"github.com/Paksonite/gitnest/internal/gitx"
)

// Icons come in two sets: plain Unicode symbols that every font has, and
// Nerd Font glyphs (config "icons: nerd") for fonts such as MesloLGS NF.
// Only glyphs from the Font Awesome, Octicons, Devicons and Seti ranges are
// used, which are present in both Nerd Fonts v2 and v3.
var nerdIcons bool

func setIcons(mode string) { nerdIcons = strings.EqualFold(mode, "nerd") }

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

// fileIcon is the icon shown before a file or directory in the Files panel,
// followed by a space; empty without Nerd Font icons.
func fileIcon(path string, dir, collapsed bool) string {
	if !nerdIcons {
		return ""
	}
	if dir {
		if collapsed {
			return nfFolder + " "
		}
		return nfFolderOpen + " "
	}
	base := strings.ToLower(filepath.Base(path))
	if ic, ok := nfByName[base]; ok {
		return ic + " "
	}
	if strings.HasPrefix(base, "dockerfile") {
		return nfDocker + " "
	}
	if ic, ok := nfByExt[filepath.Ext(base)]; ok {
		return ic + " "
	}
	return nfFile + " "
}

// refGlyph is the icon for a reflog action.
func refGlyph(k gitx.RefKind) string {
	if nerdIcons {
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
	if !nerdIcons {
		return ""
	}
	if remote {
		return nfRemoteBranch + " "
	}
	return nfBranchIcon + " "
}
