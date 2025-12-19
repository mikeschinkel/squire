package retinue

// OutputFormat defines the output format for module listing
type OutputFormat string

const (
	TableOutputFormat OutputFormat = "table"
	JSONOutputFormat  OutputFormat = "json"
	CSVOutputFormat   OutputFormat = "csv"
)

// String returns the string representation of OutputFormat
func (f OutputFormat) String() string {
	return string(f)
}

// IsValid checks if the output format is valid
func (f OutputFormat) IsValid() bool {
	switch f {
	case TableOutputFormat, JSONOutputFormat, CSVOutputFormat:
		return true
	default:
		return false
	}
}
