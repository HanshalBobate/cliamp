package cmd

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"cliamp/applog"
	"cliamp/config"
	"cliamp/theme"
)

// Customise opens the customisation TUI.
func Customise() (bool, error) {
	c, err := config.LoadCustomisations()
	if err != nil {
		return false, fmt.Errorf("customise: loading existing customisations: %w", err)
	}

	cm := newCustomiseModel(c)
	p := tea.NewProgram(cm)
	m, err := p.Run()
	if err != nil {
		return false, fmt.Errorf("customise: running TUI: %w", err)
	}
	cmOut, ok := m.(*customiseModel)
	if ok && cmOut.err != nil {
		return false, fmt.Errorf("customise: saving customisations: %w", cmOut.err)
	}
	return ok && cmOut.saved, nil
}

type customiseModel struct {
	c       config.Customisations
	cursor  int
	w, h    int
	saved   bool
	err     error
	fields  []customField
	activeT theme.Theme
}

type customField struct {
	label string
	key   string
	val   string
	help  string
}

func newCustomiseModel(c config.Customisations) *customiseModel {
	themes := theme.LoadAll()
	var active theme.Theme
	if len(themes) > 0 {
		active = themes[0] // fallback
	}
	// Try to get active theme from config
	cfg, err := config.Load()
	if err != nil {
		applog.Info("customise: failed to load config for theme: %v", err)
	}
	for _, t := range themes {
		if strings.EqualFold(t.Name, cfg.Theme) {
			active = t
			break
		}
	}

	hideLocalStr := "false"
	if c.HideLocal {
		hideLocalStr = "true"
	}

	fields := []customField{
		{
			label: "Hide Local Provider",
			key:   "hide_local",
			val:   hideLocalStr,
			help:  "Type 'true' to hide the Local provider in the SRC list.",
		},
		{
			label: "Radio Provider Name",
			key:   "radio_name",
			val:   c.ProviderNames["radio"],
			help:  "Rename 'Radio' to any arbitrary string.",
		},
		{
			label: "Radio Playlist Replacement",
			key:   "radio_url",
			val:   c.RadioReplacement,
			help:  "A YouTube Playlist URL to scrape and replace the Radio default.",
		},
	}

	for i, ep := range c.ExtraPlaylists {
		fields = append(fields, customField{
			label: fmt.Sprintf("Extra Playlist %d Name", i+1),
			key:   fmt.Sprintf("ep_name_%d", i),
			val:   ep.Name,
			help:  "Name of the playlist.",
		})
		fields = append(fields, customField{
			label: fmt.Sprintf("Extra Playlist %d URL", i+1),
			key:   fmt.Sprintf("ep_url_%d", i),
			val:   ep.URL,
			help:  "YouTube playlist URL.",
		})
	}

	fields = append(fields, customField{
		label: "New Playlist Name",
		key:   "ep_name_new",
		val:   "",
		help:  "Name for an additional YouTube playlist.",
	})
	fields = append(fields, customField{
		label: "New Playlist URL",
		key:   "ep_url_new",
		val:   "",
		help:  "URL for an additional YouTube playlist.",
	})

	return &customiseModel{
		c:       c,
		activeT: active,
		fields:  fields,
	}
}

func (m *customiseModel) Init() tea.Cmd {
	return func() tea.Msg { return tea.RequestWindowSize() }
}

func (m *customiseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m, nil

	case tea.PasteMsg:
		content := strings.Map(func(r rune) rune {
			if r == '\n' || r == '\r' || r == '\t' {
				return -1
			}
			return r
		}, msg.Content)
		m.fields[m.cursor].val += content
		return m, nil

	case tea.KeyPressMsg:
		switch msg.Code {
		case tea.KeyEscape:
			return m, tea.Quit
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
		case tea.KeyTab, tea.KeyDown:
			if msg.Mod&tea.ModShift != 0 {
				if m.cursor > 0 {
					m.cursor--
				}
			} else {
				if m.cursor < len(m.fields)-1 {
					m.cursor++
				}
			}
		case tea.KeyEnter:
			if m.cursor < len(m.fields)-1 {
				m.cursor++
			} else {
				return m.save()
			}
		case tea.KeyBackspace:
			cur := m.fields[m.cursor].val
			if cur != "" {
				m.fields[m.cursor].val = removeLastRune(cur)
			}
		case tea.KeySpace:
			m.fields[m.cursor].val += " "
		default:
			if s := msg.String(); s == "ctrl+s" || s == "ctrl+d" {
				return m.save()
			}
			if len(msg.Text) > 0 {
				m.fields[m.cursor].val += msg.Text
			}
		}
	}
	return m, nil
}

func (m *customiseModel) save() (tea.Model, tea.Cmd) {
	if strings.ToLower(strings.TrimSpace(m.fields[0].val)) == "true" {
		m.c.HideLocal = true
	} else {
		m.c.HideLocal = false
	}
	m.c.ProviderNames["radio"] = m.fields[1].val
	m.c.RadioReplacement = m.fields[2].val

	var extras []config.CustomPlaylist
	for i := 3; i+1 < len(m.fields); i += 2 {
		name := strings.TrimSpace(m.fields[i].val)
		url := strings.TrimSpace(m.fields[i+1].val)

		if name != "" || url != "" {
			if name == "" {
				name = "Custom Playlist"
			}
			if url != "" {
				extras = append(extras, config.CustomPlaylist{
					Name: name,
					URL:  url,
				})
			}
		}
	}
	m.c.ExtraPlaylists = extras

	m.err = config.SaveCustomisations(m.c)
	if m.err == nil {
		m.saved = true
	}
	return m, tea.Quit
}

func (m *customiseModel) View() tea.View {
	if m.err != nil {
		return tea.NewView(fmt.Sprintf("Error saving: %v\n\nPress any key to exit.", m.err))
	}

	var b strings.Builder
	b.WriteString("\n  cliamp --customise\n\n")

	for i, f := range m.fields {
		label := f.label
		if i == m.cursor {
			label = "> " + label
			if m.activeT.Accent != "" {
				label = lipgloss.NewStyle().Foreground(lipgloss.Color(m.activeT.Accent)).Render(label)
			}
		} else {
			label = "  " + label
		}

		b.WriteString(label + "\n")

		val := f.val
		if val == "" {
			val = "(empty)"
			if m.activeT.FG != "" {
				val = lipgloss.NewStyle().Foreground(lipgloss.Color(m.activeT.FG)).Render(val)
			}
		}

		borderStyle := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			PaddingLeft(1)
		if m.activeT.FG != "" {
			borderStyle = borderStyle.BorderForeground(lipgloss.Color(m.activeT.FG))
		}
		border := borderStyle.Render(val)

		b.WriteString("  " + border + "\n")

		helpStyle := lipgloss.NewStyle()
		if m.activeT.FG != "" {
			helpStyle = helpStyle.Foreground(lipgloss.Color(m.activeT.FG))
		}
		help := helpStyle.Render("  " + f.help)
		b.WriteString(help + "\n\n")
	}

	b.WriteString("  [Enter] Next/Save  [Tab] Next  [Esc] Cancel\n")

	view := tea.NewView(b.String())
	view.AltScreen = true
	return view
}
