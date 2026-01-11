package utils

// PrintCurlImportUsage prints the usage help for the curl import feature
func PrintCurlImportUsage() {
	PrintWarning("cURL Import Usage:")
	_ = WriteCommandHelp([]*CommandHelp{
		{
			Command:     "hulak import curl 'curl https://example.com'",
			Description: "Import from command-line argument",
		},
		{
			Command:     "echo 'curl ...' | hulak import curl",
			Description: "Import from pipe",
		},
		{
			Command:     "hulak import -o ./my-api.hk.yaml curl 'curl ...'",
			Description: "Specify output file with -o flag",
		},
		{
			Command:     "hulak import curl <<'EOF'\\n curl ...  EOF",
			Description: "Import from heredoc (best for DevTools paste)",
		},
	})
	PrintInfo("Learn more: hulak help")
}
