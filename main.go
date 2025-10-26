// main.go - 主程序入口
package main

import (
	"LLM-Automation-Test/config"
	"LLM-Automation-Test/report"
	"fmt"
)

func main() {
	fmt.Println("🚗 购车决策大模型自动化评测系统（Gemini 驱动）")

	cfg, err := config.Load()
	if err != nil {
		fmt.Println(err.Error())
		fmt.Println("按回车键退出...")
		_, _ = fmt.Scanln() // 等待用户输入
		return
	}

	initialPrompt := "我最近打算买车，预算大约20万元左右，主要用于城市通勤，希望空间舒适、安全性高。"

	// 测试 Qwen 和 DeepSeek
	models := []string{cfg.DeepSeek.Model, cfg.Qwen.Model}
	for _, model := range models {
		fmt.Printf("\n\n=== 测试模型: %s ===\n", model)
		res := report.Run(cfg, model, initialPrompt)
		report.SaveReport(model, res)
		report.SaveMarkdownReport(model, res)
	}

	fmt.Println("\n✅ 所有测试完成！")

	fmt.Println("按回车键退出...")
	_, _ = fmt.Scanln() // 等待用户输入
	return
}
