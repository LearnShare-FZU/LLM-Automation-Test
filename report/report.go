// report/report.go - 执行对话并产出报告
package report

import (
	"LLM-Automation-Test/config"
	"LLM-Automation-Test/drivers"
	"LLM-Automation-Test/targets"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Turn struct {
	Round     int
	UserInput string
	AIReply   string
}

type TestResult struct {
	ModelName      string    `json:"model_name"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	TotalRounds    int       `json:"total_rounds"`
	Conversation   []Turn    `json:"conversation"`
	FinalRecommend string    `json:"final_recommendation"`
	Success        bool      `json:"success"`
	GeminiJudgeLog string    `json:"gemini_judge_log"`
}

func Run(cfg *config.Config, modelName string, initialPrompt string) *TestResult {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout*time.Duration(cfg.MaxRounds))
	defer cancel()

	result := &TestResult{
		ModelName:    modelName,
		StartTime:    time.Now(),
		Conversation: []Turn{},
		Success:      false,
	}

	userInput := initialPrompt
	fmt.Printf("👤 用户初始输入: %s\n", userInput)

	messages := []targets.Message{{Role: "user", Content: userInput}}

	for round := 1; round <= cfg.MaxRounds; round++ {
		var (
			aiReply string
			err     error
		)
		if modelName == cfg.Qwen.Model {
			aiReply, err = targets.CallOpenAICompat(ctx, cfg.Qwen.APIKey, cfg.Qwen.BaseURL, modelName, messages, cfg)
		} else if modelName == cfg.DeepSeek.Model {
			aiReply, err = targets.CallOpenAICompat(ctx, cfg.DeepSeek.APIKey, cfg.DeepSeek.BaseURL, modelName, messages, cfg)
		} else {
			err = fmt.Errorf("未知模型: %s", modelName)
		}

		if err != nil {
			fmt.Printf("⚠️ Target AI 调用失败: %v\n", err)
			break
		}

		fmt.Printf("🤖 %s 回复: %s\n", modelName, aiReply)
		result.Conversation = append(result.Conversation, Turn{
			Round:     round,
			UserInput: userInput,
			AIReply:   aiReply,
		})

		var convHistory []map[string]string
		for _, t := range result.Conversation {
			convHistory = append(convHistory, map[string]string{
				"user": t.UserInput,
				"ai":   t.AIReply,
			})
		}

		isComplete, reason, err := drivers.CheckIfDecisionComplete(ctx, convHistory, cfg)
		if err != nil {
			fmt.Printf("⚠️ Gemini 判定失败: %v\n", err)
			os.Exit(-1)
		} else if isComplete {
			result.FinalRecommend = aiReply
			result.Success = true
			result.TotalRounds = round
			result.GeminiJudgeLog = reason
			fmt.Printf("✅ Gemini 判断完成: %s\n", reason)
			break
		}

		fmt.Printf("ℹ️ Gemini 判断未完成: %s\n", reason)

		history := fmt.Sprintf("用户: %s\nAI: %s", userInput, aiReply)
		nextQ, err := drivers.GenerateNextUserQuestion(ctx, history, cfg)
		if err != nil {
			nextQ = "请继续提供更合适的推荐方式。"
		}
		userInput = nextQ
		fmt.Printf("➡️ 开始第 %d 轮...\n", round+1)
		fmt.Printf("👤 Gemini 生成用户问题: %s\n", userInput)

		messages = append(messages,
			targets.Message{Role: "assistant", Content: aiReply},
			targets.Message{Role: "user", Content: userInput},
		)

		time.Sleep(300 * time.Millisecond)
	}

	result.EndTime = time.Now()
	if !result.Success {
		result.TotalRounds = cfg.MaxRounds
	}
	return result
}

// SaveReport 保存原始 JSON 报告
func SaveReport(outputDir, modelName string, runIndex int, result *TestResult) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		fmt.Printf("⚠️ 无法创建报告目录: %v\n", err)
		return
	}

	filename := filepath.Join(outputDir, fmt.Sprintf("report_run%d_%d.json", runIndex, time.Now().UnixNano()))
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("⚠️ 无法保存 JSON 报告: %v\n", err)
		return
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	enc.Encode(result)
	fmt.Printf("✅ JSON 报告已保存: %s\n", filename)
}

// SaveMarkdownReport 保存 Markdown 版对话记录
func SaveMarkdownReport(outputDir, modelName string, runIndex int, result *TestResult) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		fmt.Printf("⚠️ 无法创建报告目录: %v\n", err)
		return
	}

	filename := filepath.Join(outputDir, fmt.Sprintf("report_run%d_%d.md", runIndex, time.Now().UnixNano()))
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("⚠️ 无法保存 Markdown 报告: %v\n", err)
		return
	}
	defer file.Close()

	fmt.Fprintf(file, "# 多轮对话测试报告\n\n")
	fmt.Fprintf(file, "- **测试模型**: `%s`\n", result.ModelName)
	fmt.Fprintf(file, "- **运行序号**: %d\n", runIndex)
	fmt.Fprintf(file, "- **开始时间**: %s\n", result.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "- **结束时间**: %s\n", result.EndTime.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "- **对话轮数**: %d\n", result.TotalRounds)
	fmt.Fprintf(file, "- **是否成功**: %v\n\n", result.Success)

	if result.Success {
		fmt.Fprintf(file, "## 🎯 最终推荐\n\n")
		fmt.Fprintf(file, "%s\n\n", result.FinalRecommend)
		if result.GeminiJudgeLog != "" {
			fmt.Fprintf(file, "> **Gemini 判定原因**: %s\n\n", result.GeminiJudgeLog)
		}
	} else {
		fmt.Fprintf(file, "## ⚠️ 未完成推荐\n\n")
		fmt.Fprintf(file, "对话达到 %d 轮后仍未完成推荐。\n\n", result.TotalRounds)
	}

	fmt.Fprintf(file, "## 💬 对话详情\n\n")

	for _, turn := range result.Conversation {
		fmt.Fprintf(file, "### 第 %d 轮\n\n", turn.Round)
		fmt.Fprintf(file, "**👤 用户**:\n\n%s\n\n", turn.UserInput)
		fmt.Fprintf(file, "**🤖 %s**:\n\n%s\n\n", result.ModelName, turn.AIReply)
		fmt.Fprintf(file, "---\n\n")
	}

	fmt.Printf("✅ Markdown 报告已保存: %s\n", filename)
}
