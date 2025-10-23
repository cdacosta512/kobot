package checks

import (
	"html/template"
	"os"
)

// WriteHTMLReport writes an HTML summary file of all namespace checks.
func WriteHTMLReport(results []PodCheckResult, totalPods, totalNamespaces, failed int) error {
	tmpl := `
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="UTF-8">
		<title>Kobot Health Check Report</title>
		<style>
			body { font-family: Arial, sans-serif; margin: 40px; color: #333; }
			h1 { color: #326CE5; }
			table { border-collapse: collapse; width: 100%; margin-top: 20px; }
			th, td { border: 1px solid #ccc; padding: 8px 12px; text-align: left; }
			th { background-color: #f2f2f2; }
			.pass { color: green; font-weight: bold; }
			.fail { color: red; font-weight: bold; }
		</style>
	</head>
	<body>
		<h1>Kobot Health Check Report</h1>
		<p><b>Total Namespaces:</b> {{.TotalNamespaces}} |
		<b>Total Pods:</b> {{.TotalPods}} |
		<b>Failed Namespaces:</b> {{.Failed}}</p>
		<table>
			<tr>
				<th>Namespace</th>
				<th>Status</th>
				<th>Pods Checked</th>
				<th>Pods Failed</th>
			</tr>
			{{range .Results}}
				<tr>
					<td>{{.Name}}</td>
					{{if gt .PodsFailed 0}}
						<td class="fail">FAIL</td>
					{{else}}
						<td class="pass">PASS</td>
					{{end}}
					<td>{{.PodsChecked}}</td>
					<td>{{.PodsFailed}}</td>
				</tr>
			{{end}}
		</table>
	</body>
	</html>
	`

	data := struct {
		Results         []PodCheckResult
		TotalPods       int
		TotalNamespaces int
		Failed          int
	}{
		Results:         results,
		TotalPods:       totalPods,
		TotalNamespaces: totalNamespaces,
		Failed:          failed,
	}

	t := template.Must(template.New("report").Parse(tmpl))
	f, err := os.Create("kobot-report.html")
	if err != nil {
		return err
	}
	defer f.Close()

	return t.Execute(f, data)
}