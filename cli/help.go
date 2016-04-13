package main

import (
	"github.com/codegangsta/cli"
)

// Override templates
func init() {
	cli.AppHelpTemplate = `Usage:
  {{if .UsageText}}{{.UsageText}}{{else}}{{.HelpName}} {{if .Flags}}[global options]{{end}}{{if .Commands}} command [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{end}}{{if .Commands}}

Commands:{{range .Categories}}{{range $index, $cmd := .Commands}}{{if .Subcommands}}{{range .Subcommands}}
  {{$cmd.Name}} {{.Name}}{{ "\t" }}{{.Usage}}{{end}}{{else}}
  {{.Name}}{{ "\t" }}{{.Usage}}{{end}}{{end}}{{end}}{{end}}{{if .Flags}}

Global Options:
   {{range .Flags}}{{.}}
   {{end}}{{end}}
`

	cli.CommandHelpTemplate = `Usage:
   {{.HelpName}}{{if .Flags}} [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}{{if .Category}}

Category:
   {{.Category}}{{end}}{{if .Description}}

Description:
   {{.Description}}{{end}}{{if .Flags}}

Options:
   {{range .Flags}}{{.}}
   {{end}}{{ end }}
`

	cli.SubcommandHelpTemplate = `Usage:
   {{.HelpName}} command{{if .Flags}} [command options]{{end}} {{if .ArgsUsage}}{{.ArgsUsage}}{{else}}[arguments...]{{end}}

Commands:{{range .Categories}}{{range .Commands}}
  {{.Name}}{{with .ShortName}}, {{.}}{{end}}{{ "\t" }}{{.Usage}}{{end}}{{end}}{{if .Flags}}

Options:
   {{range .Flags}}{{.}}
   {{end}}{{end}}
`
}
