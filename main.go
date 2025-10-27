// main.go - 多轮对话批次测试入口
package main

import (
	"LLM-Automation-Test/config"
	"LLM-Automation-Test/report"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	fmt.Println("欢迎使用多轮对话自动化测试系统 - Gemini 驱动")

	cfg, err := config.Load()
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println("加载配置失败，按回车退出...")
		_, _ = fmt.Scanln()
		return
	}

	initialPrompt := "预算二十万左右，希望能买到通过性好、安全性高的新能源 SUV，适合家庭出行。"

	targetModels := collectTargetModels(cfg)
	if len(targetModels) == 0 {
		fmt.Println("未配置 Target 模型信息，无法执行测试。按回车退出...")
		_, _ = fmt.Scanln()
		return
	}

	fmt.Printf("共需测试模型 %d 个，每个模型重复 %d 次。\n", len(targetModels), cfg.TestRuns)

	modelResults := make(map[string][]*report.TestResult)

	for _, model := range targetModels {
		modelDir, err := ensureModelDir(model)
		if err != nil {
			fmt.Printf("⚠️ 无法创建模型目录 %s: %v\n", model, err)
			continue
		}

		fmt.Printf("\n\n=== 测试模型: %s ===\n", model)
		var runs []*report.TestResult
		var runAnalyses []*report.QuantAnalysis
		for runIndex := 1; runIndex <= cfg.TestRuns; runIndex++ {
			fmt.Printf("\n-- 第 %d 次会话测试 --\n", runIndex)
			res := report.Run(cfg, model, initialPrompt)
			report.SaveReport(modelDir, model, runIndex, res)
			report.SaveMarkdownReport(modelDir, model, runIndex, res)

			analysis := report.AnalyzeResult(res)
			report.PrintAnalysisSummary(runIndex, analysis)

			runs = append(runs, res)
			runAnalyses = append(runAnalyses, analysis)
		}

		batch := report.AnalyzeBatch(model, runs)
		report.SaveQuantSummary(modelDir, runAnalyses, batch)
		report.PrintBatchSummary(batch)

		modelResults[model] = runs
	}

	fmt.Println("\n📊 生成模型间对比...\n")
	comparison := report.CompareBatchResults(modelResults)
	report.PrintComparisonSummary(comparison)
	report.SaveComparisonReport(comparison)

	fmt.Println("\n全部批次测试完成，按回车退出程序...")
	_, _ = fmt.Scanln()
}

func collectTargetModels(cfg *config.Config) []string {
	models := []string{}
	if cfg.DeepSeek.Model != "" {
		models = append(models, cfg.DeepSeek.Model)
	}
	if cfg.Qwen.Model != "" {
		models = append(models, cfg.Qwen.Model)
	}
	return models
}

func ensureModelDir(modelName string) (string, error) {
	dir := filepath.Join("report", sanitizeModelName(modelName))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func sanitizeModelName(name string) string {
	replacer := strings.NewReplacer(
		" ", "_",
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(name)
}
