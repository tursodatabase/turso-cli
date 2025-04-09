package cmd

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tursodatabase/turso-cli/internal/turso"
)

type PageFetcher interface {
	FetchPage(pageSize int, cursor *string) (turso.ListResponse, error)
}

type dbListModel struct {
	databases []turso.Database
	page      int
	pageSize  int
	fetcher   PageFetcher
	loading   bool
	cursor    *string
	hasMore   bool
	err       error
}

func (m dbListModel) Init() tea.Cmd {
	return m.fetchNextPage
}

func fetchPage(fetcher PageFetcher, pageSize int, cursor *string) (dbPageMsg, error) {
	r, err := fetcher.FetchPage(pageSize, cursor)
	if err != nil {
		return dbPageMsg{}, err
	}

	var nextCursor *string = nil
	if r.Pagination != nil && r.Pagination.Next != nil {
		nextCursor = r.Pagination.Next
	}

	return dbPageMsg{
		databases: r.Databases,
		cursor:    nextCursor,
	}, nil
}

func (m dbListModel) fetchNextPage() tea.Msg {
	r, err := fetchPage(m.fetcher, m.pageSize, m.cursor)
	if err != nil {
		return errMsg{err}
	}
	return r
}

type dbPageMsg struct {
	databases []turso.Database
	cursor    *string
}

type errMsg struct {
	err error
}

func (m dbListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "n", " ", "enter":
			if m.hasMore && !m.loading {
				m.loading = true
				return m, m.fetchNextPage
			}
		}

	case dbPageMsg:
		m.loading = false
		m.databases = msg.databases
		m.page++

		if msg.cursor != nil {
			m.hasMore = true
			m.cursor = msg.cursor
			return m, nil
		} else {
			m.hasMore = false
			return m, tea.Quit
		}

	case errMsg:
		m.err = msg.err
		return m, tea.Quit
	}

	return m, nil
}

func formatGroup(group string) string {
	if group == "" {
		return "-"
	}
	return group
}

func (m dbListModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	var headers []string
	var data [][]string

	for _, database := range m.databases {
		row := []string{database.Name}
		row = append(row, formatGroup(database.Group))
		row = append(row, getDatabaseUrl(&database))
		data = append(data, row)
	}

	headers = append(headers, "NAME")
	headers = append(headers, "GROUP")
	headers = append(headers, "URL")

	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range data {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	var s strings.Builder

	if len(data) > 0 {
		for i, h := range headers {
			if i > 0 {
				s.WriteString("    ")
			}
			s.WriteString(fmt.Sprintf("%-*s", widths[i], h))
		}
		s.WriteString("\n")
	}

	for _, row := range data {
		for i, cell := range row {
			if i > 0 {
				s.WriteString("    ")
			}
			s.WriteString(fmt.Sprintf("%-*s", widths[i], cell))
		}
		s.WriteString("\n")
	}

	if m.hasMore {
		s.WriteString("\nPress 'n' or Enter for next page (q to quit)")
	}

	return s.String()
}

func printDatabaseList(fetcher PageFetcher) error {
	if !isInteractive() {
		var allDatabases []turso.Database
		var cursor *string

		for {
			r, err := fetchPage(fetcher, 100, cursor)
			if err != nil {
				return err
			}

			allDatabases = append(allDatabases, r.databases...)
			cursor = r.cursor
			if cursor == nil {
				break
			}
		}

		model := dbListModel{
			databases: allDatabases,
		}

		fmt.Print(model.View())
		return nil
	}

	pageSize := 1
	_, height, err := term.GetSize(int(os.Stdout.Fd()))
	if err == nil && height > 4 {
		pageSize = height - 3
	}

	model := dbListModel{
		pageSize: pageSize,
		fetcher:  fetcher,
	}

	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running database list: %w", err)
	}

	return nil
}
