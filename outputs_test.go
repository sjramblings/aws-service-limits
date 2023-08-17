package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrintHeaderTable(t *testing.T) {
	// Redirect stdout to a buffer
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set outputFormat to "table"
	outputFormat = "table"

	// Call the function
	printHeader()

	// Restore stdout
	w.Close()
	os.Stdout = old

	// Get the output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	// Assert the expected output
	expectedOutput := "Account ID   Region          Service              Global Value           Usage           Quota Name                                                                      \n"
	assert.Equal(t, expectedOutput, buf.String())
}

func TestPrintQuotaMarkdown(t *testing.T) {
	// Redirect stdout to a buffer
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set outputFormat to "table"
	outputFormat = "markdown"

	// Call the function
	printQuota("123456789", "us-west-1", "EC2", "false", "100", "50", "Attachments per VPC")

	// Restore stdout
	w.Close()
	os.Stdout = old

	// Get the output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	// Assert the expected output
	expectedOutput := "| 123456789 | us-west-1 | EC2 | false | 100 | 50 | Attachments per VPC |\n"
	assert.Equal(t, expectedOutput, buf.String())
}

func TestPrintQuotaCsv(t *testing.T) {
	// Redirect stdout to a buffer
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set outputFormat to "table"
	outputFormat = "csv"

	// Call the function
	printQuota("123456789", "us-west-1", "EC2", "false", "100", "50", "Attachments per VPC")

	// Restore stdout
	w.Close()
	os.Stdout = old

	// Get the output
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	// Assert the expected output
	expectedOutput := "123456789,us-west-1,EC2,false,100,50,Attachments per VPC\n"
	assert.Equal(t, expectedOutput, buf.String())
}

func TestIsOutputRedirected(t *testing.T) {
	// Call the function
	isRedirected := isOutputRedirected()

	// Assert the expected result
	assert.True(t, isRedirected)
}
