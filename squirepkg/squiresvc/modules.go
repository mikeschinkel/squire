package squiresvc

import (
	"encoding/csv"
	"encoding/json/jsontext"
	jsonv2 "encoding/json/v2"
	"io"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// Modules is a slice of Module pointers
type Modules []*Module

// JSON returns JSON representation of modules
func (ms Modules) JSON() (jsonText string) {
	var data []byte
	var err error

	data, err = jsonv2.Marshal(ms, jsontext.WithIndent("  "))
	if err != nil {
		jsonText = "[]"
		goto end
	}

	jsonText = string(data)

end:
	return jsonText
}

// TableWriter returns a configured table.Writer for pretty printing the module list
func (ms Modules) TableWriter() (tw table.Writer) {
	var m *Module
	var row table.Row

	tw = table.NewWriter()

	if len(ms) > 0 {
		// Build header row
		tw.AppendHeader(table.Row{
			"REPO ROOT",
			"REL DIR",
			"MODULE PATH",
			"KIND",
			"VERSIONED",
			"REQUIRES",
		})

		// Build data rows
		for _, m = range ms {
			if m == nil {
				continue
			}

			row = table.Row{
				m.RepoRoot,
				m.RelDir,
				m.ModulePath,
				m.Kind.String(),
				formatBool(m.Versioned),
				formatRequires(m.Requires),
			}
			tw.AppendRow(row)
		}
	}

	// Configure column alignments
	tw.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, Align: text.AlignLeft},   // REPO ROOT
		{Number: 2, Align: text.AlignLeft},   // REL DIR
		{Number: 3, Align: text.AlignLeft},   // MODULE PATH
		{Number: 4, Align: text.AlignLeft},   // KIND
		{Number: 5, Align: text.AlignCenter}, // VERSIONED
		{Number: 6, Align: text.AlignLeft},   // REQUIRES
	})

	// Set style
	tw.SetStyle(table.StyleLight)

	return tw
}

// CSV writes modules as CSV to the provided writer
func (ms Modules) CSV(w io.Writer) (err error) {
	var csvWriter *csv.Writer
	var m *Module

	csvWriter = csv.NewWriter(w)

	// Write header
	err = csvWriter.Write([]string{
		"repo_root",
		"rel_dir",
		"module_path",
		"kind",
		"versioned",
		"requires",
	})
	if err != nil {
		goto end
	}

	// Write rows
	for _, m = range ms {
		if m == nil {
			continue
		}

		err = csvWriter.Write([]string{
			string(m.RepoRoot),
			string(m.RelDir),
			string(m.ModulePath),
			m.Kind.String(),
			formatBool(m.Versioned),
			formatRequires(m.Requires),
		})
		if err != nil {
			goto end
		}
	}

	csvWriter.Flush()
	err = csvWriter.Error()

end:
	return err
}

// formatBool converts boolean to string
func formatBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// formatRequires formats the requires slice as a comma-separated string
func formatRequires(requires []ModulePath) string {
	var result ModulePath
	var i int

	if len(requires) == 0 {
		return ""
	}

	for i = range requires {
		if i > 0 {
			result += ", "
		}
		result += requires[i]
	}

	return string(result)
}
