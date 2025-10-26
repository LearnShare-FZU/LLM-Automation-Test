// drivers/gemini.go - Gemini 作为 Driver AI（用户模拟 + 决策判断）
package drivers

import (
	"LLM-Automation-Test/config"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type GeminiMessage struct {
	Role  string `json:"role"`
	Parts []struct {
		Text string `json:"text"`
	} `json:"parts"`
}

type GeminiRequest struct {
	Contents []GeminiMessage `json:"contents"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

type DecisionCheckResponse struct {
	DecisionComplete bool   `json:"decision_complete"`
	Reason           string `json:"reason"`
}

func newHTTPClientWithProxy(proxyStr string) *http.Client {
	if proxyStr == "" {
		return &http.Client{Timeout: 60 * time.Second}
	}
	parsed, err := url.Parse(proxyStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ GEMINI_PROXY 格式错误: %v\n", err)
		return &http.Client{Timeout: 60 * time.Second}
	}
	transport := &http.Transport{Proxy: http.ProxyURL(parsed)}
	return &http.Client{Timeout: 60 * time.Second, Transport: transport}
}

func callGeminiRaw(ctx context.Context, prompt string, cfg *config.Config) (string, error) {

	reqBody := GeminiRequest{
		Contents: []GeminiMessage{{
			Role: "user",
			Parts: []struct {
				Text string `json:"text"`
			}([]struct{ Text string }{{Text: prompt}}),
		}},
	}
	body, _ := json.Marshal(reqBody)

	client := newHTTPClientWithProxy(cfg.GeminiProxy)
	req, _ := http.NewRequestWithContext(ctx, "POST", cfg.Gemini.BaseURL, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Gemini error %d: %s", resp.StatusCode, string(b))
	}

	var result GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response")
	}
	return result.Candidates[0].Content.Parts[0].Text, nil
}

// GenerateNextUserQuestion 生成下一轮用户提问
func GenerateNextUserQuestion(ctx context.Context, history string, cfg *config.Config) (string, error) {
	prompt := fmt.Sprintf(`你正在扮演一个有目的的购车客户，目标是最终让AI（你正在对话的对象）提供一个**附带充分理由的最终购车推荐**。

请根据当前的对话历史（如下方所示），判断对话进展到哪个阶段，并提出一个**自然、具体**且能推动决策进程的下一轮问题。

**对话目标阶段指导：**
1.  **初步筛选：** 询问初步推荐。
2.  **细化需求：** 根据初步推荐，提出具体的偏好（如保值率、油耗、预算上限、颜色等）。
3.  **结构化信息：** 在AI提供多个选择后，要求提供对比参数表格。
4.  **最终决策：** 询问最终推荐的车型及理由。

当前对话历史：
%s

**输出要求：**
-   **只输出问题本身**。
-   不得包含任何解释、分析、引导语或标点符号（如引号）。
-   请确保问题清晰且推进对话进入下一个目标阶段。
`, history)
	resp, err := callGeminiRaw(ctx, prompt, cfg)
	if err != nil {
		return "", err
	}
	resp = strings.TrimSpace(resp)
	resp = strings.Trim(resp, `"“”‘’`)
	return resp, nil
}

// CheckIfDecisionComplete 判断购车决策是否已完成
func CheckIfDecisionComplete(ctx context.Context, conversation []map[string]string, cfg *config.Config) (bool, string, error) {
	var sb strings.Builder
	for _, turn := range conversation {
		sb.WriteString(fmt.Sprintf("【用户】%s\n【AI】%s\n\n", turn["user"], turn["ai"]))
	}
	prompt := fmt.Sprintf(`
你是一位专注于对话系统评测的专家。请根据以下严格标准，分析并判断提供的“购车决策”对话是否已成功结束。

	**任务完成的严格标准：**
	1.  **明确选择：** AI 必须在对话的最后一步明确且只推荐了**一款**具体的、可购买的车型名称。
	2.  **充分理由：** AI 提供的理由必须全面涵盖用户在整个对话中提及的所有关键需求（包括但不限于：预算、通勤、空间、安全、保值率、油耗等）。
	3.  **决策终结：** 用户在获得AI的最终推荐后，应该无需再提出任何后续问题即可做出购买决策。

	当前对话记录：
	%s

	**输出格式要求（极度严格）：**
	-   你**必须**以纯粹的、无前缀、无解释的 JSON 格式输出结果。
-   **禁止**输出任何 Markdown 代码块标记或其他文字。
	-   JSON 结构必须是：
		{"decision_complete": true/false, "reason": "在此提供简要的判断说明和未完成的原因（如果适用）"}
	
`, sb.String())

	respText, err := callGeminiRaw(ctx, prompt, cfg)
	if err != nil {
		return false, "", err
	}
	respText = strings.TrimSpace(respText)
	respText = strings.TrimPrefix(respText, "```json")
	respText = strings.TrimSuffix(respText, "```")

	var result DecisionCheckResponse
	if err := json.Unmarshal([]byte(respText), &result); err != nil {
		return false, fmt.Sprintf("格式错误: %s", respText), nil
	}
	return result.DecisionComplete, result.Reason, nil
}
