package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Paksonite/gitnest/internal/gitx"
)

// The commit graph is drawn the way lazygit does it: one row per commit,
// every line of development in its own column and colour, merges and
// branch points joined with box-drawing lines on the commit's own row.

// gcell is one character of a graph row and the lane whose colour it takes
// (-1 for none).
type gcell struct {
	ch    rune
	color int
}

const (
	nodeCommit = '○'
	nodeMerge  = '◎'
)

// buildGraph lays out commits (newest first, topological order) and returns
// a row of cells per commit. Each column is two cells wide: the lane's
// symbol and the connector to its right.
func buildGraph(cs []gitx.Commit) [][]gcell {
	var lanes []string // commit each column is waiting for; "" = free
	var colors []int   // colour of each column
	next := 0
	newColor := func() int { c := next; next++; return c }
	free := func(ls []string, from int) int {
		for k := from; k < len(ls); k++ {
			if ls[k] == "" {
				return k
			}
		}
		return -1
	}

	rows := make([][]gcell, len(cs))
	for i, c := range cs {
		col := -1
		for k, h := range lanes {
			if h == c.Hash {
				col = k
				break
			}
		}
		if col < 0 { // a branch tip nobody pointed at yet
			col = free(lanes, 0)
			if col < 0 {
				col = len(lanes)
				lanes, colors = append(lanes, ""), append(colors, 0)
			}
			lanes[col], colors[col] = c.Hash, newColor()
		}
		var joins []int // other columns ending here (children whose parent this is)
		for k, h := range lanes {
			if k != col && h == c.Hash {
				joins = append(joins, k)
			}
		}

		after := append([]string(nil), lanes...)
		afterColors := append([]int(nil), colors...)
		for _, k := range joins {
			after[k] = ""
		}
		after[col] = ""
		if len(c.Parents) > 0 {
			after[col] = c.Parents[0]
		}
		var links []int // columns this merge's other parents continue in
		newLane := map[int]bool{}
		for _, p := range c.Parents[min(1, len(c.Parents)):] {
			k := -1
			for j, h := range after {
				if h == p {
					k = j
					break
				}
			}
			if k < 0 {
				k = free(after, col+1)
				if k < 0 {
					k = len(after)
					after, afterColors = append(after, ""), append(afterColors, 0)
				}
				after[k], afterColors[k] = p, newColor()
				newLane[k] = true
			}
			links = append(links, k)
		}

		width := max(len(lanes), len(after))
		row := make([]gcell, 2*width)
		for k := range row {
			row[k] = gcell{' ', -1}
		}
		set := func(k int, ch rune, color int) { row[2*k] = gcell{ch, color} }
		for k := 0; k < len(lanes); k++ {
			if lanes[k] != "" && k != col {
				set(k, '│', colors[k])
			}
		}
		node := nodeCommit
		if c.Merge {
			node = nodeMerge
		}
		set(col, node, colors[col])

		// Horizontal lines from the node to every column it connects with.
		lo, hi := col, col
		ends := map[int]rune{}
		endColor := map[int]int{}
		for _, k := range joins {
			ends[k] = map[bool]rune{true: '╯', false: '╰'}[k > col]
			endColor[k] = colors[k]
		}
		for _, k := range links {
			switch {
			case newLane[k]:
				ends[k] = map[bool]rune{true: '╮', false: '╭'}[k > col]
			case k > col:
				ends[k] = '┤'
			default:
				ends[k] = '├'
			}
			endColor[k] = afterColors[k]
		}
		for k := range ends {
			lo, hi = min(lo, k), max(hi, k)
		}
		// A segment takes the colour of the outermost column on its side.
		leftColor, rightColor := -1, -1
		if lo < col {
			leftColor = endColor[lo]
		}
		if hi > col {
			rightColor = endColor[hi]
		}
		for k := lo; k < hi; k++ {
			segColor := rightColor
			if k < col {
				segColor = leftColor
			}
			if k != col {
				if ch, ok := ends[k]; ok {
					if k != lo { // the line goes on past this column
						ch = '┼'
					}
					set(k, ch, endColor[k])
				} else if row[2*k].ch == '│' {
					row[2*k].ch = '┼'
				} else {
					set(k, '─', segColor)
				}
			}
			row[2*k+1] = gcell{'─', segColor}
		}
		if ch, ok := ends[hi]; ok && hi != col {
			set(hi, ch, endColor[hi])
		}

		// Trailing free columns are dropped from the lane list.
		for len(after) > 0 && after[len(after)-1] == "" {
			after, afterColors = after[:len(after)-1], afterColors[:len(afterColors)-1]
		}
		lanes, colors = after, afterColors
		// Trim trailing blanks so rows don't carry empty columns.
		end := len(row)
		for end > 0 && row[end-1].ch == ' ' {
			end--
		}
		rows[i] = row[:end]
	}
	return rows
}

// laneColor picks a colour for a graph lane from the theme.
func laneColor(i int) lipgloss.TerminalColor {
	palette := cAuthors
	if len(palette) == 0 {
		palette = []lipgloss.TerminalColor{cCyan, cYellow, cGreen, cRed, cAccent}
	}
	return palette[i%len(palette)]
}

// graphText renders a graph row as plain and coloured text.
func graphText(row []gcell) (plain, styled string) {
	var p, st strings.Builder
	for _, c := range row {
		p.WriteRune(c.ch)
		if c.color < 0 || c.ch == ' ' {
			st.WriteRune(c.ch)
			continue
		}
		st.WriteString(fg(laneColor(c.color), string(c.ch)))
	}
	return p.String(), st.String()
}

// graphString is the plain graph, used by tests.
func graphString(rows [][]gcell) string {
	var out []string
	for _, r := range rows {
		p, _ := graphText(r)
		out = append(out, p)
	}
	return strings.Join(out, "\n")
}
