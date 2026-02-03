package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"google.golang.org/api/drive/v3"
)

type OutputFormat string

const (
	FormatDetailed OutputFormat = "detailed"
	FormatTable    OutputFormat = "table"
)

var (
	debugMode    bool
	outputFormat OutputFormat = FormatDetailed
)

func SetDebug(debug bool) {
	debugMode = debug
}

func SetOutputFormat(format OutputFormat) {
	outputFormat = format
}

func GetOutputFormat() OutputFormat {
	return outputFormat
}

func Debug(format string, a ...interface{}) {
	if debugMode {
		fmt.Printf("[DEBUG] "+format+"\n", a...)
	}
}

func ConfirmAction(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(prompt)
		input, _ := reader.ReadString('\n')
		input = strings.ToLower(strings.TrimSpace(input))
		if input == "y" || input == "yes" {
			return true
		}
		if input == "n" || input == "no" || input == "" {
			return false
		}
		fmt.Println("Invalid input. Please enter 'y' or 'n'.")
	}
}

// ExternalFileData holds data for external file shares
type ExternalFileData struct {
	FileName   string
	DriveName  string
	OwnerName  string
	FolderName string
	Link       string
	SharedWith string
	Type       string
	Role       string
}

// FoundFileData holds data for found files
type FoundFileData struct {
	FileName         string
	DriveName        string
	FolderName       string
	Link             string
	PermissionID     string
	PermissionType   string
	PermissionTarget string
}

var externalFilesData []ExternalFileData
var foundFilesData []FoundFileData

func ResetExternalFilesData() {
	externalFilesData = []ExternalFileData{}
}

func ResetFoundFilesData() {
	foundFilesData = []FoundFileData{}
}

func AddExternalFile(f *drive.File, driveName, ownerName, folderName string, p *drive.Permission) {
	externalFilesData = append(externalFilesData, ExternalFileData{
		FileName:   f.Name,
		DriveName:  driveName,
		OwnerName:  ownerName,
		FolderName: folderName,
		Link:       f.WebViewLink,
		SharedWith: p.EmailAddress,
		Type:       p.Type,
		Role:       p.Role,
	})
}

func AddFoundFile(f *drive.File, driveName, folderName, permissionIdentifier string, p *drive.Permission) {
	foundFilesData = append(foundFilesData, FoundFileData{
		FileName:         f.Name,
		DriveName:        driveName,
		FolderName:       folderName,
		Link:             f.WebViewLink,
		PermissionID:     p.Id,
		PermissionType:   p.Type,
		PermissionTarget: permissionIdentifier,
	})
}

func PrintExternalFile(f *drive.File, driveName, ownerName, folderName string, p *drive.Permission) {
	if outputFormat == FormatTable {
		AddExternalFile(f, driveName, ownerName, folderName, p)
	} else {
		fmt.Printf("\n\033[31m[EXTERNAL]\033[0m File: %s\n", f.Name)
		fmt.Printf("Drive: %s | Owner: %s | Folder: %s\n", driveName, ownerName, folderName)
		fmt.Printf("Link: %s\n", f.WebViewLink)
		fmt.Printf("Shared with: %s (Type: %s, Role: %s)\n", p.EmailAddress, p.Type, p.Role)
	}
}

func PrintFoundFile(f *drive.File, driveName, folderName, permissionIdentifier string, p *drive.Permission) {
	if outputFormat == FormatTable {
		AddFoundFile(f, driveName, folderName, permissionIdentifier, p)
	} else {
		fmt.Printf("\n[FOUND] File: %s\n", f.Name)
		fmt.Printf("Drive: %s | Folder: %s\n", driveName, folderName)
		fmt.Printf("Link: %s\n", f.WebViewLink)
		fmt.Printf("Permission ID: %s (Type: %s, Target: %s)\n", p.Id, p.Type, permissionIdentifier)
	}
}

func PrintExternalFilesTable() {
	if len(externalFilesData) == 0 {
		return
	}

	fmt.Println("\n" + strings.Repeat("=", 150))
	fmt.Println("EXTERNAL SHARES - TABLE VIEW")
	fmt.Println(strings.Repeat("=", 150))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "FILE NAME\tDRIVE\tOWNER\tFOLDER\tSHARED WITH\tTYPE\tROLE\tLINK")
	fmt.Fprintln(w, strings.Repeat("-", 150))

	for _, data := range externalFilesData {
		// Truncate long names for better table display
		fileName := truncateString(data.FileName, 40)
		driveName := truncateString(data.DriveName, 25)
		ownerName := truncateString(data.OwnerName, 20)
		folderName := truncateString(data.FolderName, 30)
		sharedWith := truncateString(data.SharedWith, 30)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			fileName, driveName, ownerName, folderName, sharedWith, data.Type, data.Role, data.Link)
	}

	w.Flush()
	fmt.Println(strings.Repeat("=", 150))
	fmt.Printf("Total: %d external shares\n", len(externalFilesData))
}

func PrintFoundFilesTable() {
	if len(foundFilesData) == 0 {
		return
	}

	fmt.Println("\n" + strings.Repeat("=", 150))
	fmt.Println("FOUND FILES - TABLE VIEW")
	fmt.Println(strings.Repeat("=", 150))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "FILE NAME\tDRIVE\tFOLDER\tPERMISSION ID\tTYPE\tTARGET\tLINK")
	fmt.Fprintln(w, strings.Repeat("-", 150))

	for _, data := range foundFilesData {
		// Truncate long names for better table display
		fileName := truncateString(data.FileName, 40)
		driveName := truncateString(data.DriveName, 25)
		folderName := truncateString(data.FolderName, 30)
		target := truncateString(data.PermissionTarget, 30)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			fileName, driveName, folderName, data.PermissionID, data.PermissionType, target, data.Link)
	}

	w.Flush()
	fmt.Println(strings.Repeat("=", 150))
	fmt.Printf("Total: %d files found\n", len(foundFilesData))
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func PrintSharedDrive(count int, d *drive.Drive) {
	fmt.Printf("[%d] Name: %s | ID: %s\n", count, d.Name, d.Id)
}
