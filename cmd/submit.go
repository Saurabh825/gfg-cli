package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Saurabh825/gfg-cli/internal/api"
	"github.com/Saurabh825/gfg-cli/internal/config"

	"github.com/spf13/cobra"
)

var submitCmd = &cobra.Command{
	Use:   "submit",
	Short: "Submit your solution to GFG.",
	Run: func(cmd *cobra.Command, args []string) {
		cwd, _ := os.Getwd()
		slug := filepath.Base(cwd)

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

		pid := getProblemId(slug)
		if pid == "" {
			fmt.Println("Error: Could not obtain problem ID.")
			return
		}

		triggerUrl := fmt.Sprintf("https://practiceapiorigin.geeksforgeeks.org/api/latest/problems/%s/submit/compile/", slug)
		payload := map[string]string{
			"source":       "https://www.geeksforgeeks.org",
			"request_type": "solutionCheck",
			"userCode":     userCode,
			"language":     apiLang,
			"pid":          pid,
		}

		body, contentType, _ := createMultipart(payload)
		fmt.Printf("Submitting solution for %s...\n", slug)

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

		var submissionId string
		if id, ok := resultsData["submission_id"].(float64); ok {
			submissionId = fmt.Sprintf("%.0f", id)
		} else if idStr, ok := resultsData["submission_id"].(string); ok {
			submissionId = idStr
		}

		if submissionId == "" {
			fmt.Println("Error: No submission_id in response.")
			return
		}

		fmt.Printf("✔ Submission triggered. submission_id: %s\n", submissionId)

		pollUrl := "https://practiceapiorigin.geeksforgeeks.org/api/latest/problems/submission/submit/result/"
		fmt.Print("Polling for submission results...")

		for i := 0; i < 60; i++ {
			time.Sleep(2 * time.Second)
			pollPayload := map[string]string{
				"sub_id":   submissionId,
				"sub_type": "solutionCheck",
				"pid":      pid,
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

			status, _ := data["status"].(string)
			if status == "SUCCESS" {
				fmt.Println("\n\n--- Submission Results ---")

				subStatus, _ := data["sub_status"].(float64)
				if subStatus == 1 {
					fmt.Println("✅ Correct Answer!")
				} else {
					viewMode, _ := data["view_mode"].(string)
					if viewMode == "" {
						viewMode = "Unknown"
					}
					fmt.Printf("❌ Result: %s\n", viewMode)
				}

				if msg, ok := data["message"].(map[string]interface{}); ok {
					if execTime, ok := msg["execution_time"].(float64); ok {
						fmt.Printf("Execution Time: %.2fs\n", execTime)
					}
					if accuracy, ok := msg["accuracy"].(float64); ok {
						fmt.Printf("Accuracy: %.0f%%\n", accuracy)
					}
				}

				if processed, ok := data["test_cases_processed"].(float64); ok {
					if total, ok := data["total_test_cases"].(float64); ok {
						fmt.Printf("Test Cases: %.0f / %.0f\n", processed, total)
					}
				}

				return
			} else if status == "error" || data["error"] != nil {
				errMsg, _ := data["message"].(string)
				if errMsg == "" {
					errMsg = "Unknown error"
				}
				fmt.Printf("\n❌ Submission Error: %s\n", errMsg)
				return
			}

			fmt.Print(".")
		}

		fmt.Println("\nTimed out waiting for submission results.")
	},
}

func init() {
	rootCmd.AddCommand(submitCmd)
}
