package utils

import (
	"regexp"
	"strings"
)

func ParseTestCasesFromMD(markdown string) []map[string]string {
	var testcases []map[string]string

	inputRe := regexp.MustCompile(`(?i)Input:\s*`)
	outputRe := regexp.MustCompile(`(?i)Output:\s*`)
	endRe := regexp.MustCompile(`(?i)(Explanation:|Example|Input:|Constraints:)`)

	inputLocs := inputRe.FindAllStringIndex(markdown, -1)

	for i := range inputLocs {
		startIdx := inputLocs[i][1]

		outLoc := outputRe.FindStringIndex(markdown[startIdx:])
		if outLoc == nil {
			continue
		}

		inputStr := strings.TrimSpace(markdown[startIdx : startIdx+outLoc[0]])
		outStartIdx := startIdx + outLoc[1]

		var outputStr string
		endLoc := endRe.FindStringIndex(markdown[outStartIdx:])
		if endLoc == nil {
			outputStr = strings.TrimSpace(markdown[outStartIdx:])
		} else {
			outputStr = strings.TrimSpace(markdown[outStartIdx : outStartIdx+endLoc[0]])
		}

		var parts []string
		var currentPart []rune
		depth := 0
		for _, char := range inputStr {
			switch char {
			case '[', '{', '(':
				depth++
			case ']', '}', ')':
				depth--
			}
			if char == ',' && depth == 0 {
				parts = append(parts, strings.TrimSpace(string(currentPart)))
				currentPart = []rune{}
			} else {
				currentPart = append(currentPart, char)
			}
		}
		if len(currentPart) > 0 {
			parts = append(parts, strings.TrimSpace(string(currentPart)))
		}

		var processedInputs []string
		for _, p := range parts {
			re := regexp.MustCompile(`^[a-zA-Z_\[\]]*\s*[:=]\s*`)
			cleanPart := strings.TrimSpace(re.ReplaceAllString(p, ""))
			if strings.Contains(cleanPart, "->") {
				items := strings.Split(cleanPart, "->")
				for j := range items {
					items[j] = strings.TrimSpace(items[j])
				}
				processedInputs = append(processedInputs, "["+strings.Join(items, ",")+"]")
			} else {
				processedInputs = append(processedInputs, cleanPart)
			}
		}

		testcases = append(testcases, map[string]string{
			"input":  strings.Join(processedInputs, "\n"),
			"output": outputStr,
		})
	}

	return testcases
}
