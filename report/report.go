// report/report.go - 测试执行与报告生成
package report

import (
	"LLM-Automation-Test/config"
	"LLM-Automation-Test/drivers"
	"LLM-Automation-Test/targets"
	"context"
	"encoding/json"
	"fmt"
	"os"
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
	fmt.Printf("👤 用户（初始）: %s\n", userInput)

	messages := []targets.Message{{Role: "user", Content: userInput}}

	for round := 1; round <= cfg.MaxRounds; round++ {

		// 调用 Target AI（Qwen 或 DeepSeek）
		var (
			aiReply string
			err     error
		)
		if modelName == cfg.Qwen.Model {
			aiReply, err = targets.CallOpenAICompat(ctx, cfg.Qwen.APIKey, cfg.Qwen.BaseURL, modelName, messages, cfg)
		} else if modelName == cfg.DeepSeek.Model {
			aiReply, err = targets.CallOpenAICompat(ctx, cfg.DeepSeek.APIKey, cfg.DeepSeek.BaseURL, modelName, messages, cfg)
		}

		if err != nil {
			fmt.Printf("❌ Target AI 调用失败: %v\n", err)
			break
		}

		fmt.Printf("🤖 %s 回复: %s\n", modelName, aiReply)
		result.Conversation = append(result.Conversation, Turn{
			Round:     round,
			UserInput: userInput,
			AIReply:   aiReply,
		})

		// 构建对话历史供 Gemini 判断
		var convHistory []map[string]string
		for _, t := range result.Conversation {
			convHistory = append(convHistory, map[string]string{
				"user": t.UserInput,
				"ai":   t.AIReply,
			})
		}

		// ✅ 由 Gemini 判断是否结束
		isComplete, reason, err := drivers.CheckIfDecisionComplete(ctx, convHistory, cfg)
		if err != nil {
			fmt.Printf("⚠️ Gemini 判断出错，继续对话: %v\n", err)
			os.Exit(-1)
		} else if isComplete {
			result.FinalRecommend = aiReply
			result.Success = true
			result.TotalRounds = round
			result.GeminiJudgeLog = reason
			fmt.Printf("✅ Gemini 判定完成（%s）\n", reason)
			break
		} else {
			fmt.Printf("⏳ Gemini 判定未完成（%s）\n", reason)
		}

		// 生成下一轮用户提问
		history := fmt.Sprintf("【用户】%s\n【AI】%s", userInput, aiReply)
		nextQ, err := drivers.GenerateNextUserQuestion(ctx, history, cfg)
		if err != nil {
			nextQ = "请给出最终推荐车型和理由。"
		}
		userInput = nextQ
		fmt.Printf("第 %d 轮开始...\n", round+1)
		fmt.Printf("👤 用户（Gemini生成）: %s\n", userInput)

		// 更新对话历史
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

func SaveReport(modelName string, result *TestResult) {
	filename := fmt.Sprintf("report_%s_%d.json", modelName, time.Now().Unix())
	file, _ := os.Create(filename)
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	enc.Encode(result)
	fmt.Printf("📄 报告已保存: %s\n", filename)
}

// SaveMarkdownReport 保存对话历史为 Markdown 格式
func SaveMarkdownReport(modelName string, result *TestResult) {
	filename := fmt.Sprintf("report_%s_%d.md", modelName, time.Now().Unix())
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("⚠️ 无法创建 Markdown 报告: %v\n", err)
		return
	}
	defer file.Close()

	// 写入标题
	fmt.Fprintf(file, "# 购车决策自动化评测报告\n\n")
	fmt.Fprintf(file, "- **测试模型**: `%s`\n", result.ModelName)
	fmt.Fprintf(file, "- **开始时间**: %s\n", result.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "- **结束时间**: %s\n", result.EndTime.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "- **总轮数**: %d\n", result.TotalRounds)
	fmt.Fprintf(file, "- **是否成功完成**: %v\n\n", result.Success)

	if result.Success {
		fmt.Fprintf(file, "## ✅ 最终推荐\n\n")
		fmt.Fprintf(file, "%s\n\n", result.FinalRecommend)
		if result.GeminiJudgeLog != "" {
			fmt.Fprintf(file, "> **Gemini 判断结束理由**: %s\n\n", result.GeminiJudgeLog)
		}
	} else {
		fmt.Fprintf(file, "## ⚠️ 未完成决策\n\n")
		fmt.Fprintf(file, "对话在 %d 轮后超时，未获得明确推荐。\n\n", result.TotalRounds)
	}

	fmt.Fprintf(file, "## 💬 完整对话记录\n\n")

	for _, turn := range result.Conversation {
		fmt.Fprintf(file, "### 第 %d 轮\n\n", turn.Round)
		fmt.Fprintf(file, "**👤 用户**:\n\n%s\n\n", turn.UserInput)
		fmt.Fprintf(file, "**🤖 %s**:\n\n%s\n\n", result.ModelName, turn.AIReply)
		fmt.Fprintf(file, "---\n\n")
	}

	fmt.Printf("📄 Markdown 报告已保存: %s\n", filename)
}
