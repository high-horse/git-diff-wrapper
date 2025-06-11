package models

type DiffFile struct {
    FileName string
    Hunks    []DiffHunk
}

type DiffHunk struct {
    Header string
    Lines  []DiffLine
}

type DiffLine struct {
    Type    string // "+", "-", or " "
    Content string
}
