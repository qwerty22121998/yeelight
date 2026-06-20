package main

import (
	"html"
	"strings"
	"sync"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/widgets"
)

// logs is the process-wide sink slog tees into (see setupLogging). The Log tab
// attaches its view to it once built.
var logs = &logSink{}

type logEntry struct {
	level string // DEBUG/INFO/WARN/ERROR, "" if the line carried no level
	text  string // full formatted slog line
}

// logSink is the io.Writer slog writes to. It keeps the most recent records in
// a ring (so the Log tab can show history from before it was opened) and pushes
// live records to the attached view, which decides whether the current filter
// shows them. ponytail: 1000-entry ring; yeelight.log holds the full history.
type logSink struct {
	mu   sync.Mutex
	buf  []logEntry
	view *logView
}

func (s *logSink) Write(p []byte) (int, error) {
	for line := range strings.SplitSeq(strings.TrimRight(string(p), "\n"), "\n") {
		e := logEntry{level: parseLevel(line), text: line}
		s.mu.Lock()
		s.buf = append(s.buf, e)
		if len(s.buf) > 1000 {
			s.buf = s.buf[len(s.buf)-1000:]
		}
		v := s.view
		s.mu.Unlock()
		if v != nil {
			ev := e
			runOnUI(func() { v.append(ev) })
		}
	}
	return len(p), nil
}

func (s *logSink) snapshot() []logEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]logEntry(nil), s.buf...)
}

func (s *logSink) clear() {
	s.mu.Lock()
	s.buf = nil
	s.mu.Unlock()
}

func (s *logSink) attach(v *logView) {
	s.mu.Lock()
	s.view = v
	s.mu.Unlock()
}

// parseLevel pulls INFO/WARN/... out of a slog TextHandler line (level=INFO).
func parseLevel(line string) string {
	_, rest, ok := strings.Cut(line, "level=")
	if !ok {
		return ""
	}
	if j := strings.IndexByte(rest, ' '); j >= 0 {
		rest = rest[:j]
	}
	return rest
}

var levelRank = map[string]int{"DEBUG": 0, "INFO": 1, "WARN": 2, "ERROR": 3}

var levelColor = map[string]string{
	"DEBUG": "#9e9e9e",
	"WARN":  "#ffb300",
	"ERROR": "#ff5252",
	// INFO left uncolored so it inherits the palette text color (light & dark).
}

type logView struct {
	edit   *widgets.QPlainTextEdit
	minLvl *widgets.QComboBox // index 0=All, 1=Debug, 2=Info, 3=Warn, 4=Error
	search *widgets.QLineEdit
	follow *widgets.QCheckBox
}

func (v *logView) matches(e logEntry) bool {
	if q := strings.ToLower(v.search.Text()); q != "" && !strings.Contains(strings.ToLower(e.text), q) {
		return false
	}
	if min := v.minLvl.CurrentIndex() - 1; min >= 0 {
		if r, ok := levelRank[e.level]; ok && r < min {
			return false
		}
	}
	return true
}

func fmtEntry(e logEntry) string {
	text := html.EscapeString(e.text)
	if c := levelColor[e.level]; c != "" {
		return `<span style="color:` + c + `">` + text + `</span>`
	}
	return text
}

// append renders one live record if it passes the current filter.
func (v *logView) append(e logEntry) {
	if !v.matches(e) {
		return
	}
	v.edit.AppendHtml(fmtEntry(e))
	if v.follow.IsChecked() {
		sb := v.edit.VerticalScrollBar()
		sb.SetValue(sb.Maximum())
	}
}

// rebuild re-renders the whole view from the ring under the current filter.
func (v *logView) rebuild() {
	v.edit.SetUpdatesEnabled(false)
	v.edit.Clear()
	for _, e := range logs.snapshot() {
		if v.matches(e) {
			v.edit.AppendHtml(fmtEntry(e))
		}
	}
	v.edit.SetUpdatesEnabled(true)
	sb := v.edit.VerticalScrollBar()
	sb.SetValue(sb.Maximum())
}

// logUI builds the Log tab: a filter toolbar over a read-only colored view.
func logUI() widgets.QWidget_ITF {
	w := widgets.NewQWidget(nil, core.Qt__Widget)
	root := widgets.NewQVBoxLayout()
	w.SetLayout(root)

	minLvl := widgets.NewQComboBox(nil)
	minLvl.AddItems([]string{"All", "Debug", "Info", "Warn", "Error"})
	search := widgets.NewQLineEdit(nil)
	search.SetPlaceholderText("Filter…")
	search.SetClearButtonEnabled(true)
	follow := widgets.NewQCheckBox2("Auto-scroll", nil)
	follow.SetChecked(true)
	clearBtn := widgets.NewQPushButton2("Clear", nil)

	bar := widgets.NewQHBoxLayout()
	bar.AddWidget(widgets.NewQLabel2("Level:", nil, 0), 0, 0)
	bar.AddWidget(minLvl, 0, 0)
	bar.AddWidget(search, 1, 0)
	bar.AddWidget(follow, 0, 0)
	bar.AddWidget(clearBtn, 0, 0)

	edit := widgets.NewQPlainTextEdit(nil)
	edit.SetReadOnly(true)
	edit.SetMaximumBlockCount(2000) // cap memory; drops oldest lines
	edit.SetLineWrapMode(widgets.QPlainTextEdit__NoWrap)
	edit.SetFont(gui.QFontDatabase_SystemFont(gui.QFontDatabase__FixedFont))

	root.AddLayout(bar, 0)
	root.AddWidget(edit, 1, 0)

	v := &logView{edit: edit, minLvl: minLvl, search: search, follow: follow}
	minLvl.ConnectCurrentIndexChanged(func(int) { v.rebuild() })
	search.ConnectTextChanged(func(string) { v.rebuild() })
	clearBtn.ConnectClicked(func(bool) {
		logs.clear()
		v.rebuild()
	})

	logs.attach(v)
	v.rebuild()
	return w
}
