package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Saurabh825/gfg-cli/internal/api"
	"github.com/Saurabh825/gfg-cli/internal/config"

	"github.com/spf13/cobra"
)

var (
	localOpt  bool
	onlineOpt bool
)

var apiLangMap = map[string]string{
	"cpp":        "cpp",
	"python":     "python3",
	"java":       "java",
	"javascript": "javascript",
}

func parseTestcases(filename string) []map[string]string {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil
	}

	blocksRaw := strings.Split(string(content), "\n\n")
	var blocks []string
	for _, b := range blocksRaw {
		if strings.TrimSpace(b) != "" {
			blocks = append(blocks, strings.TrimSpace(b))
		}
	}
	if len(blocks) == 0 && strings.TrimSpace(string(content)) != "" {
		blocks = []string{strings.TrimSpace(string(content))}
	}

	var testCases []map[string]string
	for _, block := range blocks {
		inMatch := regexp.MustCompile(`(?is)input:\n(.*?)(\noutput:|$)`).FindStringSubmatch(block)
		outMatch := regexp.MustCompile(`(?is)output:\n(.*?)$`).FindStringSubmatch(block)

		if len(inMatch) > 1 && len(outMatch) > 1 {
			rawLines := strings.Split(strings.TrimSpace(inMatch[1]), "\n")
			expected := strings.TrimSpace(outMatch[1])
			var cleanLines []string
			for _, line := range rawLines {
				cl := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(line, "[", ""), "]", ""), ",", " ")
				cleanLines = append(cleanLines, cl)
			}
			testCases = append(testCases, map[string]string{
				"input":  strings.Join(cleanLines, "\n"),
				"output": expected,
			})
		}
	}
	return testCases
}

func runLocalTest(lang string) {
	ext := extMap[lang]
	if ext == "" {
		ext = "cpp"
	}
	solFile := fmt.Sprintf("solution.%s", ext)

	if _, err := os.Stat(solFile); os.IsNotExist(err) {
		fmt.Printf("Error: %s not found.\n", solFile)
		return
	}

	cases := parseTestcases("testcases.txt")
	if len(cases) == 0 {
		fmt.Println("Error: No test cases found in testcases.txt")
		return
	}

	fmt.Printf("Running local tests for %s...\n", lang)
	var executable []string
	if lang == "cpp" {
		fmt.Println("Compiling solution.cpp...")
		cmd := exec.Command("g++", "-O3", solFile, "-o", "solution_bin")
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Compilation Failed:\n%s\n", output)
			return
		}
		executable = []string{"./solution_bin"}
	} else if lang == "python" {
		executable = []string{"python3", solFile}
	} else if lang == "java" {
		fmt.Println("Compiling solution.java...")
		cmd := exec.Command("javac", solFile)
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Compilation Failed:\n%s\n", output)
			return
		}
		executable = []string{"java", "Solution"}
	} else if lang == "javascript" {
		executable = []string{"node", solFile}
	}

	passed := 0
	for i, c := range cases {
		fullInput := fmt.Sprintf("1\n%s\n", c["input"])
		cmd := exec.Command(executable[0], executable[1:]...)
		cmd.Stdin = strings.NewReader(fullInput)
		out, err := cmd.CombinedOutput()

		if err != nil && out == nil {
			fmt.Printf("Case %d: ERROR - %v\n", i+1, err)
			continue
		}

		rawActual := strings.Split(strings.TrimSpace(string(out)), "\n")
		var actualLines []string
		for _, l := range rawActual {
			l = strings.TrimSpace(l)
			if l != "" && l != "~" {
				actualLines = append(actualLines, l)
			}
		}
		actualOutput := strings.Join(actualLines, "\n")

		rawExpected := strings.Split(c["output"], "\n")
		var expectedLines []string
		for _, l := range rawExpected {
			l = strings.TrimSpace(l)
			if l != "" {
				expectedLines = append(expectedLines, l)
			}
		}
		expectedNormalized := strings.Join(expectedLines, "\n")

		if actualOutput == expectedNormalized {
			fmt.Printf("Case %d: PASSED ✅\n", i+1)
			passed++
		} else {
			fmt.Printf("Case %d: FAILED ❌\n", i+1)
			fmt.Printf("   Input: \n%s\n", c["input"])
			fmt.Printf("   Expected: \n%s\n", expectedNormalized)
			fmt.Printf("   Actual:   \n%s\n", actualOutput)
		}
	}
	fmt.Println(strings.Repeat("-", 30))
	fmt.Printf("Total: %d/%d passed.\n", passed, len(cases))
}

func getProblemId(slug string) string {
	meta, err := api.FetchMetaInfo(slug)
	if err != nil {
		return ""
	}
	if results, ok := meta["results"].(map[string]interface{}); ok {
		if id, ok := results["id"].(float64); ok {
			return fmt.Sprintf("%.0f", id)
		}
		if id, ok := results["problem_id"].(float64); ok {
			return fmt.Sprintf("%.0f", id)
		}
		if extra, ok := results["extra"].(map[string]interface{}); ok {
			if id, ok := extra["problem_id"].(float64); ok {
				return fmt.Sprintf("%.0f", id)
			}
		}
	}
	return ""
}

func createMultipart(fields map[string]string) (*bytes.Buffer, string, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for k, v := range fields {
		writer.WriteField(k, v)
	}
	err := writer.Close()
	return body, writer.FormDataContentType(), err
}

func formatOutput(raw interface{}) []string {
	if raw == nil {
		return nil
	}
	var results []string
	switch val := raw.(type) {
	case []interface{}:
		for _, item := range val {
			if slice, ok := item.([]interface{}); ok {
				var inner []string
				for _, i := range slice {
					inner = append(inner, strings.TrimSpace(fmt.Sprint(i)))
				}
				results = append(results, strings.Join(inner, "\n"))
			} else {
				results = append(results, strings.TrimSpace(fmt.Sprint(item)))
			}
		}
	case string:
		if strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]") {
			var parsed []interface{}
			if err := json.Unmarshal([]byte(strings.ReplaceAll(val, "'", "\"")), &parsed); err == nil {
				return formatOutput(parsed)
			}
		}
		blocks := strings.Split(val, "~")
		for _, b := range blocks {
			var clean []string
			for _, l := range strings.Split(b, "\n") {
				if strings.TrimSpace(l) != "" {
					clean = append(clean, strings.TrimSpace(l))
				}
			}
			if len(clean) > 0 {
				results = append(results, strings.Join(clean, "\n"))
			}
		}
	}
	var final []string
	for _, r := range results {
		if r != "" && r != "~" {
			final = append(final, r)
		}
	}
	return final
}

func onlineTest(slug string) {
	langPreference := config.Cfg.Language
	ext := extMap[langPreference]
	if ext == "" {
		ext = "cpp"
	}
	apiLang := apiLangMap[langPreference]
	if apiLang == "" {
		apiLang = "cpp"
	}
	solFile := fmt.Sprintf("solution.%s", ext)

	content, err := os.ReadFile(solFile)
	if err != nil {
		fmt.Printf("Error: %s not found.\n", solFile)
		return
	}

	commentChar := "//"
	if ext == "py" {
		commentChar = "#"
	}
	pattern := regexp.MustCompile(`(?is)` + commentChar + `\s*@gfg\s*code=begin\s*\n(.*?)` + commentChar + `\s*@gfg\s*code=end`)
	match := pattern.FindStringSubmatch(string(content))
	if len(match) < 2 {
		fmt.Printf("Error: Could not find @gfg code tags in %s\n", solFile)
		return
	}
	userCode := strings.TrimSpace(match[1])

	cases := parseTestcases("testcases.txt")
	if len(cases) == 0 {
		fmt.Println("Error: No test cases found.")
		return
	}

	var inputs []string
	for _, c := range cases {
		inputs = append(inputs, c["input"])
	}
	customInput := fmt.Sprintf("%d\n%s", len(cases), strings.Join(inputs, "\n"))

	triggerUrl := fmt.Sprintf("https://practiceapiorigin.geeksforgeeks.org/api/latest/problems/%s/compile-sub-id/", slug)
	payload := map[string]string{
		"source":          "https://www.geeksforgeeks.org",
		"request_type":    "compileOutput",
		"input":           customInput,
		"userCode":        userCode,
		"language":        apiLang,
		"test_case_count": fmt.Sprint(len(cases)),
	}

	body, contentType, _ := createMultipart(payload)
	fmt.Printf("Triggering online test for %s...\n", slug)

	req, _ := http.NewRequest("POST", triggerUrl, body)
	for k, v := range api.GetHeaders() {
		req.Header.Set(k, v)
	}
	req.Header.Set("Content-Type", contentType)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("Error: Server returned %d\n", resp.StatusCode)
		return
	}

	respBody, _ := io.ReadAll(resp.Body)
	var resJson map[string]interface{}
	json.Unmarshal(respBody, &resJson)

	resultsData, ok := resJson["results"].(map[string]interface{})
	if !ok {
		resultsData = resJson
	}

	var subId string
	if id, ok := resultsData["sub_id"].(float64); ok {
		subId = fmt.Sprintf("%.0f", id)
	} else if id, ok := resultsData["expected_submission_id"].(float64); ok {
		subId = fmt.Sprintf("%.0f", id)
	}

	var testSolSubId string
	if id, ok := resultsData["testSolution_sub_id"].(float64); ok {
		testSolSubId = fmt.Sprintf("%.0f", id)
	} else if idStr, ok := resultsData["testSolution_submission_id"].(string); ok {
		testSolSubId = idStr
	}

	if subId == "" {
		fmt.Println("Error: No sub_id in response.")
		return
	}

	pid := getProblemId(slug)
	if pid == "" {
		fmt.Println("Error: Could not obtain problem ID.")
		return
	}

	pollUrl := "https://practiceapiorigin.geeksforgeeks.org/api/latest/problems/submission/compile-output"
	fmt.Print("Polling for results...")

	hasExpected := false
	hasActual := false
	var finalExpected, finalActual map[string]interface{}

	for i := 0; i < 40; i++ {
		time.Sleep(2 * time.Second)
		pollPayload := map[string]string{
			"sub_id":                  subId,
			"testSolution_sub_id":     testSolSubId,
			"sub_type":                "compileOutput",
			"pid":                     pid,
			"hasExpectedOutputResult": fmt.Sprint(hasExpected),
			"hasTestSolutionResult":   fmt.Sprint(hasActual),
		}

		pBody, pContentType, _ := createMultipart(pollPayload)
		pReq, _ := http.NewRequest("POST", pollUrl, pBody)
		for k, v := range api.GetHeaders() {
			pReq.Header.Set(k, v)
		}
		pReq.Header.Set("Content-Type", pContentType)

		pResp, err := client.Do(pReq)
		if err != nil {
			continue
		}
		pb, _ := io.ReadAll(pResp.Body)
		pResp.Body.Close()

		var pollResult map[string]interface{}
		json.Unmarshal(pb, &pollResult)

		data, ok := pollResult["results"].(map[string]interface{})
		if !ok {
			data = pollResult
		}

		if expected, ok := data["expectedOutput"].(map[string]interface{}); ok && expected["status"] == "SUCCESS" {
			hasExpected = true
			finalExpected = expected
		}
		if actual, ok := data["testSolution"].(map[string]interface{}); ok {
			if actual["status"] == "SUCCESS" {
				hasActual = true
				finalActual = actual
			} else if msg, mOk := actual["message"].(map[string]interface{}); mOk && msg["error"] != nil {
				hasActual = true
				finalActual = actual
			}
		}

		isFinished := (hasExpected && hasActual) || (hasActual && data["expectedOutput"] == nil)

		if isFinished {
			fmt.Println("\n--- Online Test Results ---")
			if finalActual != nil {
				if msg, ok := finalActual["message"].(map[string]interface{}); ok && msg["error"] != nil {
					fmt.Printf("❌ Error:\n%s\n", msg["error"])
					return
				}
			}

			var expMsg, actMsg map[string]interface{}
			if finalExpected != nil {
				expMsg, _ = finalExpected["message"].(map[string]interface{})
			}
			if finalActual != nil {
				actMsg, _ = finalActual["message"].(map[string]interface{})
			}

			var expOutRaw, actOutRaw interface{}
			if expMsg != nil {
				expOutRaw = expMsg["output"]
			}
			if actMsg != nil {
				actOutRaw = actMsg["your_output"]
				if actOutRaw == nil {
					actOutRaw = actMsg["output"]
				}
			}

			expOut := formatOutput(expOutRaw)
			actOut := formatOutput(actOutRaw)

			for idx, actVal := range actOut {
				statusIcon := ""
				label := fmt.Sprintf("Case %d", idx+1)
				expVal := ""

				if idx < len(expOut) {
					expVal = expOut[idx]
				} else if idx < len(cases) {
					expVal = strings.TrimSpace(cases[idx]["output"])
				}

				if expVal != "" {
					var eLines []string
					for _, l := range strings.Split(expVal, "\n") {
						if strings.TrimSpace(l) != "" {
							eLines = append(eLines, strings.TrimSpace(l))
						}
					}
					expNorm := strings.Join(eLines, "\n")

					var aLines []string
					for _, l := range strings.Split(actVal, "\n") {
						if strings.TrimSpace(l) != "" {
							aLines = append(aLines, strings.TrimSpace(l))
						}
					}
					actNorm := strings.Join(aLines, "\n")

					if actNorm == expNorm {
						statusIcon = "✅"
					} else {
						statusIcon = "❌"
					}
					expVal = expNorm
				}

				fmt.Printf("\n%s: %s\n", label, statusIcon)
				if expVal != "" {
					fmt.Printf("Expected:\n%s\n", expVal)
				}
				fmt.Printf("Got:\n%s\n", actVal)
			}
			return
		}
		fmt.Print(".")
	}
	fmt.Println("\nTimed out waiting for test results.")
}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test your solution locally or online.",
	Run: func(cmd *cobra.Command, args []string) {
		cwd, _ := os.Getwd()
		slug := filepath.Base(cwd)

		if !localOpt && !onlineOpt {
			localOpt = true
			onlineOpt = true
		}

		if localOpt {
			runLocalTest(config.Cfg.Language)
		}

		if onlineOpt {
			if config.Cfg.CookieString == "" && config.Cfg.SessionID == "" {
				fmt.Println("Warning: No authentication set. Use 'gfg config --cookie-string'.")
			}
			onlineTest(slug)
		}
	},
}

func init() {
	testCmd.Flags().BoolVarP(&localOpt, "local", "L", false, "Run local tests")
	testCmd.Flags().BoolVarP(&onlineOpt, "online", "O", false, "Run online tests")
	rootCmd.AddCommand(testCmd)
}
