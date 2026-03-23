package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Saurabh825/gfg-cli/internal/api"
	"github.com/Saurabh825/gfg-cli/internal/config"
	"github.com/Saurabh825/gfg-cli/internal/utils"

	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/spf13/cobra"
)

var langMap = map[string]string{
	"cpp":        "cpp",
	"python":     "python3",
	"javascript": "javascript",
	"c":          "c",
	"java":       "java",
}

var extMap = map[string]string{
	"cpp":        "cpp",
	"python":     "py",
	"javascript": "js",
	"c":          "c",
	"java":       "java",
}

var pickCmd = &cobra.Command{
	Use:   "pick [problem_id]",
	Short: "Pick a problem from GFG and download it locally.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		problemID := args[0]
		fmt.Printf("Searching for problem: %s...\n", problemID)

		details, err := api.FetchProblemDetails(problemID)
		if err != nil {
			fmt.Printf("Error fetching details: %v\n", err)
			return
		}

		metaInfo, err := api.FetchMetaInfo(problemID)
		if err != nil {
			fmt.Printf("Error fetching meta info: %v\n", err)
			return
		}

		results, _ := metaInfo["results"].(map[string]interface{})
		extra, _ := results["extra"].(map[string]interface{})

		lang := config.Cfg.Language
		apiLang, ok := langMap[lang]
		if !ok {
			apiLang = "cpp"
		}
		ext, ok := extMap[lang]
		if !ok {
			ext = "cpp"
		}

		targetDir := filepath.Join(lang, problemID)
		os.MkdirAll(targetDir, 0o755)

		title, _ := details["problem_name"].(string)
		if title == "" {
			title = problemID
		}
		htmlQuestion, _ := details["problem_question"].(string)

		markdownContent, _ := htmltomarkdown.ConvertString(htmlQuestion)
		markdownFull := fmt.Sprintf("# %s\n\n%s", title, markdownContent)

		os.WriteFile(filepath.Join(targetDir, "problem.md"), []byte(markdownFull), 0o644)

		initialUserFunc, _ := extra["initial_user_func"].(map[string]interface{})
		langData, ok := initialUserFunc[apiLang].(map[string]interface{})
		if !ok {
			var availableLangs []string
			for k := range initialUserFunc {
				availableLangs = append(availableLangs, k)
			}
			if len(availableLangs) > 0 {
				fmt.Printf("Warning: Preferred language '%s' not found. Available: %v\n", lang, availableLangs)
				apiLang = availableLangs[0]
				langData = initialUserFunc[apiLang].(map[string]interface{})
				for k, v := range langMap {
					if v == apiLang {
						ext = extMap[k]
						break
					}
				}
				fmt.Printf("Falling back to: %s\n", apiLang)
			}
		}

		if langData != nil {
			initial, _ := langData["initial_code"].(string)
			userStub, _ := langData["user_code"].(string)
			commentStart := "//"
			if ext == "py" {
				commentStart = "#"
			}
			taggedUserStub := fmt.Sprintf("%s @gfg code=begin\n%s\n%s @gfg code=end", commentStart, strings.TrimSpace(userStub), commentStart)

			var fullCode string
			if strings.Contains(initial, "int main()") {
				parts := strings.Split(initial, "int main()")
				fullCode = parts[0] + "\n" + taggedUserStub + "\n\nint main()" + parts[1]
			} else if strings.Contains(initial, "def main():") {
				parts := strings.Split(initial, "def main():")
				fullCode = parts[0] + "\n" + taggedUserStub + "\n\ndef main():" + parts[1]
			} else if strings.Contains(initial, "public static void main") {
				parts := strings.Split(initial, "public static void main")
				fullCode = parts[0] + "\n" + taggedUserStub + "\n\n    public static void main" + parts[1]
			} else {
				fullCode = taggedUserStub + "\n\n" + initial
			}

			os.WriteFile(filepath.Join(targetDir, fmt.Sprintf("solution.%s", ext)), []byte(fullCode), 0o644)
		} else {
			fmt.Println("Error: Could not find solution stub in API response.")
		}

		testcasesFromMd := utils.ParseTestCasesFromMD(markdownFull)
		firstOutput := "?"
		if len(testcasesFromMd) > 0 {
			firstOutput = testcasesFromMd[0]["output"]
		}

		rawApiInput, _ := extra["input"].(string)
		if rawApiInput != "" {
			delimiter := "&!//!&"
			var tcContent string
			if strings.Contains(rawApiInput, delimiter) {
				parts := strings.Split(rawApiInput, delimiter)
				for i, p := range parts {
					tcContent += "input:\n"
					cleanP := strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(p), "->", " "), ",", " ")
					tcContent += cleanP + "\n"
					tcContent += "output:\n"
					if i < len(testcasesFromMd) {
						tcContent += testcasesFromMd[i]["output"] + "\n"
					} else if i == 0 {
						tcContent += firstOutput + "\n"
					} else {
						tcContent += "?\n"
					}
					if i < len(parts)-1 {
						tcContent += "\n"
					}
				}
			} else {
				tcContent += "input:\n"
				tcContent += strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(rawApiInput), "->", " "), ",", " ") + "\n"
				tcContent += "output:\n"
				tcContent += firstOutput + "\n"
			}
			os.WriteFile(filepath.Join(targetDir, "testcases.txt"), []byte(tcContent), 0o644)
		} else {
			var tcContent string
			for i, tc := range testcasesFromMd {
				tcContent += "input:\n" + tc["input"] + "\n"
				tcContent += "output:\n" + tc["output"] + "\n"
				if i < len(testcasesFromMd)-1 {
					tcContent += "\n"
				}
			}
			os.WriteFile(filepath.Join(targetDir, "testcases.txt"), []byte(tcContent), 0o644)
		}

		fmt.Printf("✔ Successfully picked problem: %s to %s\n", problemID, targetDir)
	},
}

func init() {
	rootCmd.AddCommand(pickCmd)
}
