package cli

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"

	"gitnotes/internal/note"
)

var errPickCanceled = errors.New("selection canceled")

const (
	pickerHeight    = 12
	codeWidth       = 24
	maxLabelW       = 24
	submittedHeader = "SUBMITTED"
	defaultNoteW    = 30
)

var (
	pickerSelStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	pickerDimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	pickerYesStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	pickerNoStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	pickerTitleStyle  = lipgloss.NewStyle().Bold(true)
	pickerHeaderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Bold(true)
	detailLabelStyle  = lipgloss.NewStyle().Bold(true)
)

type pickRow struct {
	idx       int
	submitted bool
	label     string
	code      string
	note      string
	search    string
}

func submitGlyph(submitted bool) string {
	if submitted {
		return "✓"
	}
	return "✗"
}

func submitStyle(submitted bool) lipgloss.Style {
	if submitted {
		return pickerYesStyle
	}
	return pickerNoStyle
}

type pickerModel struct {
	title    string
	input    textinput.Model
	rows     []pickRow
	sources  []string
	matches  []int
	cursor   int
	labelW   int
	width    int
	chosen   int
	canceled bool
}

func pickRows(entries []note.Entry) ([]pickRow, int) {
	rows := make([]pickRow, len(entries))
	labelW := len("LOCATION")
	for i, e := range entries {
		label := e.Location().Label()
		rows[i] = pickRow{
			idx:       i,
			submitted: e.Submitted,
			label:     label,
			code:      preview(e.Code),
			note:      preview(e.Note),
			search:    strings.ToLower(label + " " + e.Note + " " + e.Code),
		}
		if w := len([]rune(label)); w > labelW {
			labelW = w
		}
	}
	if labelW > maxLabelW {
		labelW = maxLabelW
	}
	return rows, labelW
}

func newPickerModel(title string, entries []note.Entry) pickerModel {
	rows, labelW := pickRows(entries)
	sources := make([]string, len(rows))
	for i, r := range rows {
		sources[i] = r.search
	}
	ti := textinput.New()
	ti.Placeholder = "type to fuzzy-search…"
	ti.Prompt = "/ "
	ti.Focus()
	m := pickerModel{
		title:   title,
		input:   ti,
		rows:    rows,
		sources: sources,
		labelW:  labelW,
		chosen:  -1,
	}
	m.matches = m.filter("")
	return m
}

func (m pickerModel) filter(q string) []int {
	q = strings.TrimSpace(strings.ToLower(q))
	if q == "" {
		all := make([]int, len(m.rows))
		for i := range m.rows {
			all[i] = i
		}
		return all
	}

	limit := len(q)*2 + 2
	type cand struct {
		i, sub, span, pos int
	}
	var cands []cand
	for i, s := range m.sources {
		if p := strings.Index(s, q); p >= 0 {
			cands = append(cands, cand{i: i, sub: 0, span: len(q), pos: p})
			continue
		}
		span, pos := tightestSubseq(s, q)
		if span >= 0 && span <= limit {
			cands = append(cands, cand{i: i, sub: 1, span: span, pos: pos})
		}
	}
	sort.SliceStable(cands, func(a, b int) bool {
		x, y := cands[a], cands[b]
		switch {
		case x.sub != y.sub:
			return x.sub < y.sub
		case x.span != y.span:
			return x.span < y.span
		case x.pos != y.pos:
			return x.pos < y.pos
		default:
			return len(m.sources[x.i]) < len(m.sources[y.i])
		}
	})
	out := make([]int, len(cands))
	for k, c := range cands {
		out[k] = c.i
	}
	return out
}

func tightestSubseq(t, q string) (int, int) {
	best, bestStart := -1, -1
	for i := 0; i < len(t); i++ {
		qi := 0
		j := i
		for j < len(t) && qi < len(q) {
			if t[j] == q[qi] {
				qi++
			}
			j++
		}
		if qi < len(q) {
			break
		}
		end := j - 1
		qi = len(q) - 1
		k := end
		for k >= i {
			if t[k] == q[qi] {
				if qi == 0 {
					break
				}
				qi--
			}
			k--
		}
		if win := end - k + 1; best == -1 || win < best {
			best, bestStart = win, k
		}
		i = k
	}
	return best, bestStart
}

func (m pickerModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = ws.Width
		return m, nil
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c", "esc":
			m.canceled = true
			return m, tea.Quit
		case "enter":
			if len(m.matches) > 0 {
				m.chosen = m.rows[m.matches[m.cursor]].idx
			}
			return m, tea.Quit
		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "ctrl+n":
			if m.cursor < len(m.matches)-1 {
				m.cursor++
			}
			return m, nil
		}
	}

	prev := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if m.input.Value() != prev {
		m.matches = m.filter(m.input.Value())
		m.cursor = 0
	}
	return m, cmd
}

func (m pickerModel) View() string {
	if m.canceled || m.chosen >= 0 {
		return ""
	}

	var b strings.Builder
	if m.title != "" {
		b.WriteString(pickerTitleStyle.Render(m.title))
		b.WriteString("\n\n")
	}
	b.WriteString(m.input.View())
	b.WriteString("\n\n")

	noteW := m.noteWidth()
	header := fmt.Sprintf("%2s  %-*s  %-*s  %-*s  %s", "#", m.labelW, "LOCATION", codeWidth, "CODE", noteW, "NOTE", submittedHeader)
	b.WriteString(pickerHeaderStyle.Render("  " + header))
	b.WriteString("\n")

	lines := 0
	if len(m.matches) == 0 {
		b.WriteString(pickerDimStyle.Render("  no matches"))
		b.WriteString("\n")
		lines++
	} else {
		start := 0
		if m.cursor >= pickerHeight {
			start = m.cursor - pickerHeight + 1
		}
		end := min(start+pickerHeight, len(m.matches))
		for i := start; i < end; i++ {
			b.WriteString(m.renderRow(m.rows[m.matches[i]], i == m.cursor))
			b.WriteString("\n")
			lines++
		}
	}
	for ; lines < pickerHeight; lines++ {
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(pickerDimStyle.Render(fmt.Sprintf("%d/%d · ↑/↓ move · enter select · esc cancel", len(m.matches), len(m.rows))))
	b.WriteString("\n")
	return b.String()
}

func (m pickerModel) noteWidth() int {
	if m.width <= 0 {
		return defaultNoteW
	}
	fixed := 12 + m.labelW + codeWidth + len(submittedHeader)
	return max(m.width-fixed, 8)
}

func (m pickerModel) renderRow(r pickRow, selected bool) string {
	noteW := m.noteWidth()
	label := fmt.Sprintf("%-*s", m.labelW, truncate(r.label, m.labelW))
	code := fmt.Sprintf("%-*s", codeWidth, truncate(r.code, codeWidth))
	note := fmt.Sprintf("%-*s", noteW, truncate(r.note, noteW))
	glyph := submitGlyph(r.submitted)
	if selected {
		line := fmt.Sprintf("%2d  %s  %s  %s  %s", r.idx, label, code, note, glyph)
		return pickerSelStyle.Render("› " + line)
	}
	line := fmt.Sprintf("%2d  %s  %s  %s  %s", r.idx, label, code, note, submitStyle(r.submitted).Render(glyph))
	return "  " + line
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}

func (a *app) runPicker(title string, entries []note.Entry) (int, bool, error) {
	res, err := tea.NewProgram(newPickerModel(title, entries)).Run()
	if err != nil {
		return 0, false, err
	}
	m, ok := res.(pickerModel)
	if !ok || m.canceled || m.chosen < 0 {
		return 0, true, nil
	}
	return m.chosen, false, nil
}

func interactive() bool {
	return isatty.IsTerminal(os.Stdin.Fd())
}

type editModel struct {
	title    string
	input    textinput.Model
	done     bool
	canceled bool
}

func newEditModel(title, initial string) editModel {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Width = 60
	ti.SetValue(initial)
	ti.CursorEnd()
	ti.Focus()
	return editModel{title: title, input: ti}
}

func (m editModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m editModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "esc", "ctrl+c":
			m.canceled = true
			return m, tea.Quit
		case "enter":
			m.done = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m editModel) View() string {
	if m.done || m.canceled {
		return ""
	}
	var b strings.Builder
	if m.title != "" {
		b.WriteString(pickerTitleStyle.Render(m.title))
		b.WriteString("\n\n")
	}
	b.WriteString(m.input.View())
	b.WriteString("\n\n")
	b.WriteString(pickerDimStyle.Render("enter save · esc cancel"))
	b.WriteString("\n")
	return b.String()
}

func (a *app) editText(title, initial string) (string, bool, error) {
	res, err := tea.NewProgram(newEditModel(title, initial)).Run()
	if err != nil {
		return "", false, err
	}
	m, ok := res.(editModel)
	if !ok || m.canceled || !m.done {
		return "", true, nil
	}
	return m.input.Value(), false, nil
}

func (a *app) selectEntry(title string, entries []note.Entry) (int, error) {
	if !interactive() {
		return 0, fmt.Errorf("specify which note by index (0-%d); interactive selection needs a terminal", len(entries)-1)
	}
	idx, canceled, err := a.runPicker(title, entries)
	if err != nil {
		return 0, err
	}
	if canceled {
		return 0, errPickCanceled
	}
	return idx, nil
}

func (a *app) chooseIndex(title string, args []string, entries []note.Entry) (int, error) {
	if arg := firstArg(args); arg != "" {
		return selectIndex(arg, len(entries))
	}
	if len(entries) == 1 {
		return 0, nil
	}
	return a.selectEntry(title, entries)
}

func (a *app) browseEntries(title string, entries []note.Entry) error {
	if !interactive() {
		a.plainList(title, entries)
		return nil
	}
	idx, canceled, err := a.runPicker(title, entries)
	if err != nil {
		return err
	}
	if canceled {
		return nil
	}
	a.renderDetail(idx, entries[idx])
	return nil
}

func (a *app) renderDetail(idx int, e note.Entry) {
	submitted := "no"
	if e.Submitted {
		submitted = "yes"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s #%d\n", detailLabelStyle.Render("Note"), idx)
	fmt.Fprintf(&b, "%s %s\n", detailLabelStyle.Render("Location: "), e.Location().Label())
	fmt.Fprintf(&b, "%s %s\n", detailLabelStyle.Render("Submitted:"), submitted)
	if strings.TrimSpace(e.Code) != "" {
		fmt.Fprintf(&b, "\n%s\n%s\n", detailLabelStyle.Render("Code:"), e.Code)
	}
	fmt.Fprintf(&b, "\n%s\n%s\n", detailLabelStyle.Render("Note:"), e.Note)
	fmt.Fprint(a.out, b.String())
}

func (a *app) plainList(title string, entries []note.Entry) {
	if title != "" {
		fmt.Fprintln(a.out, title)
	}
	_, labelW := pickRows(entries)
	noteW := len("NOTE")
	for _, e := range entries {
		if w := len([]rune(preview(e.Note))); w > noteW {
			noteW = w
		}
	}
	fmt.Fprintf(a.out, "%2s  %-*s  %-*s  %-*s  %s\n", "#", labelW, "LOCATION", codeWidth, "CODE", noteW, "NOTE", submittedHeader)
	for i, e := range entries {
		label := fmt.Sprintf("%-*s", labelW, truncate(e.Location().Label(), labelW))
		code := fmt.Sprintf("%-*s", codeWidth, truncate(preview(e.Code), codeWidth))
		note := fmt.Sprintf("%-*s", noteW, preview(e.Note))
		fmt.Fprintf(a.out, "%2d  %s  %s  %s  %s\n", i, label, code, note, submitGlyph(e.Submitted))
	}
}
