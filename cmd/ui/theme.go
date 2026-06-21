package main

import (
	"strings"

	"github.com/therecipe/qt/widgets"
)

// Theme is the user-editable color palette. Five colors drive the whole UI; the
// rest of the look (hover/pressed/disabled states) is derived from them in the
// stylesheet. Persisted via config.toml.
type Theme struct {
	Background string // window base
	Surface    string // panels, inputs, buttons
	Text       string // all text
	Accent     string // highlights, selected states
	Border     string // outlines, slider grooves
}

func darkTheme() Theme {
	return Theme{
		Background: "#1e1f26",
		Surface:    "#262832",
		Text:       "#e4e6eb",
		Accent:     "#6c8cff",
		Border:     "#3a3d4a",
	}
}

func lightTheme() Theme {
	return Theme{
		Background: "#f4f5f7",
		Surface:    "#ffffff",
		Text:       "#1e1f26",
		Accent:     "#3a6ea5",
		Border:     "#d4d7de",
	}
}

// qApp is the running application, set once in main, used to re-apply the look
// live when the user changes style or palette.
var qApp *widgets.QApplication

// platformDefaultStyle is the style Qt picked before we touched anything,
// captured in main. "Default" in the picker (Style == "") maps back to it so
// the choice reverts live — Qt has no way to un-set a style once applied.
var platformDefaultStyle string

// applyAppearance installs the chosen Qt style (s.Style; empty = the platform
// default) then the palette stylesheet on top. The stylesheet is re-applied
// after the style swap so the custom look survives a style change.
func applyAppearance(s *Setting) {
	if qApp == nil {
		return
	}
	style := s.Style
	if style == "" {
		style = platformDefaultStyle
	}
	if style != "" {
		widgets.QApplication_SetStyle2(style)
	}
	qApp.SetStyleSheet(s.Theme.stylesheet())
}

// stylesheet renders the QSS template with the palette's colors substituted in.
func (t Theme) stylesheet() string {
	return strings.NewReplacer(
		"@bg", t.Background,
		"@surface", t.Surface,
		"@text", t.Text,
		"@accent", t.Accent,
		"@border", t.Border,
	).Replace(styleTemplate)
}

// styleTemplate is the global Qt stylesheet (QSS — CSS for Qt). @-tokens are
// replaced with the active palette. Modern look: rounded surfaces, an accent
// for interactive/selected states, derived hover/pressed/disabled styling.
// Widgets that set their own stylesheet (color swatch buttons, the method
// pills) override only the properties they name and keep the rest.
const styleTemplate = `
* {
    font-family: "Inter", "Segoe UI", "Helvetica Neue", sans-serif;
    font-size: 13px;
    color: @text;
}

QMainWindow, QDialog, QWidget { background-color: @bg; }
QToolTip { background-color: @surface; color: @text; border: 1px solid @border; border-radius: 6px; padding: 4px 8px; }
QLabel { background: transparent; }

/* Tabs */
QTabWidget::pane { border: 1px solid @border; border-radius: 10px; background: @surface; }
QTabBar::tab { background: transparent; color: @text; padding: 8px 16px; border: none; border-radius: 8px; margin: 2px; }
QTabBar::tab:hover { background: @surface; }
QTabBar::tab:selected { color: #ffffff; background: @accent; }

/* Group boxes */
QGroupBox { background-color: @surface; border: 1px solid @border; border-radius: 12px; margin-top: 14px; padding: 14px 12px 12px 12px; font-weight: 600; }
QGroupBox::title { subcontrol-origin: margin; subcontrol-position: top left; left: 12px; padding: 0 6px; color: @accent; }

/* Buttons */
QPushButton { background-color: @surface; color: @text; border: 1px solid @border; border-radius: 8px; padding: 7px 14px; }
QPushButton:hover { border-color: @accent; }
QPushButton:pressed { background-color: @accent; color: #ffffff; }
QPushButton:checked { background-color: @accent; color: #ffffff; border-color: @accent; }
QPushButton:disabled { color: @border; background-color: @bg; }

/* Sliders */
QSlider::groove:horizontal { height: 6px; background: @border; border-radius: 3px; }
QSlider::sub-page:horizontal { background: @accent; border-radius: 3px; }
QSlider::handle:horizontal { background: @text; width: 16px; margin: -6px 0; border-radius: 8px; }
QSlider::handle:horizontal:hover { background: @accent; }

/* Check / radio */
QCheckBox, QRadioButton { spacing: 8px; background: transparent; }
QCheckBox::indicator, QRadioButton::indicator { width: 18px; height: 18px; border: 1px solid @border; background: @surface; }
QCheckBox::indicator { border-radius: 5px; }
QRadioButton::indicator { border-radius: 9px; }
QCheckBox::indicator:checked { background: @accent; border-color: @accent; }
QRadioButton::indicator:checked { background: @accent; border: 4px solid @surface; }

/* Text inputs */
QLineEdit, QSpinBox, QComboBox, QPlainTextEdit { background-color: @surface; color: @text; border: 1px solid @border; border-radius: 8px; padding: 6px 10px; selection-background-color: @accent; selection-color: #ffffff; }
QLineEdit:focus, QSpinBox:focus, QComboBox:focus, QPlainTextEdit:focus { border-color: @accent; }
QComboBox::drop-down { border: none; width: 22px; }
QComboBox QAbstractItemView { background: @surface; border: 1px solid @border; border-radius: 8px; selection-background-color: @accent; outline: none; }
QSpinBox::up-button, QSpinBox::down-button { background: transparent; border: none; width: 18px; }

/* Progress bar */
QProgressBar { background-color: @surface; border: none; border-radius: 6px; max-height: 8px; text-align: center; color: transparent; }
QProgressBar::chunk { background-color: @accent; border-radius: 6px; }

/* Scrollbars */
QScrollBar:vertical { background: transparent; width: 10px; margin: 2px; }
QScrollBar:horizontal { background: transparent; height: 10px; margin: 2px; }
QScrollBar::handle:vertical { background: @border; border-radius: 5px; min-height: 24px; }
QScrollBar::handle:horizontal { background: @border; border-radius: 5px; min-width: 24px; }
QScrollBar::handle:hover { background: @accent; }
QScrollBar::add-line, QScrollBar::sub-line { width: 0; height: 0; background: none; }
QScrollBar::add-page, QScrollBar::sub-page { background: none; }

/* Containers */
QScrollArea { border: none; background: transparent; }
QSplitter::handle { background: @border; }
QSplitter::handle:horizontal { width: 2px; }
QSplitter::handle:hover { background: @accent; }

/* Status bar */
QStatusBar { background: @bg; color: @text; }
QStatusBar::item { border: none; }
`
