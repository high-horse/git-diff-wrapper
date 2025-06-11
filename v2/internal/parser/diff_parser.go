package parser

import (
    "strings"
    "go-diff/internal/models"
)

func ParseGitDiff(raw string) []models.DiffFile {
    var files []models.DiffFile
    var currentFile *models.DiffFile
    var currentHunk *models.DiffHunk

    lines := strings.Split(raw, "\n")
    for _, line := range lines {
        if strings.HasPrefix(line, "diff --git") {
            if currentFile != nil {
                files = append(files, *currentFile)
            }
            currentFile = &models.DiffFile{FileName: parseFileName(line)}
            currentHunk = nil
        } else if strings.HasPrefix(line, "@@") && currentFile != nil {
            if currentHunk != nil {
                currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
            }
            currentHunk = &models.DiffHunk{Header: line}
        } else if currentHunk != nil {
            currentHunk.Lines = append(currentHunk.Lines, models.DiffLine{
                Type:    lineType(line),
                Content: line,
            })
        }
    }

    if currentHunk != nil && currentFile != nil {
        currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
    }

    if currentFile != nil {
        files = append(files, *currentFile)
    }

    return files
}

func parseFileName(line string) string {
    parts := strings.Split(line, " ")
    if len(parts) >= 3 {
        return strings.TrimPrefix(parts[2], "b/")
    }
    return "unknown"
}

func lineType(line string) string {
    if len(line) == 0 {
        return " "
    }
    switch line[0] {
    case '+':
        return "+"
    case '-':
        return "-"
    default:
        return " "
    }
}
