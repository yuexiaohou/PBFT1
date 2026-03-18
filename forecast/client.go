package forecast

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ForecastRequest 请求参数
type ForecastRequest struct {
	InputCSV  string `json:"input_csv"`
	TargetCol string `json:"target_col"`
	Horizon   int    `json:"horizon"`
}

// ForecastData 对应单条预测结果
type ForecastData struct {
	Date          string  `json:"ds"`
	BaselineMean  float64 `json:"baseline_mean"`
	HybridPred    float64 `json:"hybrid_pred"`
	HybridLower95 float64 `json:"hybrid_lower_95"`
	HybridUpper95 float64 `json:"hybrid_upper_95"`
}

// ForecastResponse 响应参数
type ForecastResponse struct {
	Status      string         `json:"status"`
	MLTrainRMSE float64        `json:"ml_train_rmse"`
	Forecast    []ForecastData `json:"forecast"`
}

// ======================= 【高亮-2026-03-18】新增交易回传请求结构体 =======================
type RecordTradeRequest struct {
	Date   string  `json:"date"`
	Price  float64 `json:"price"`
	Amount int     `json:"amount"`
}

// Client 预测服务客户端
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient 初始化客户端
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GetPriceForecast 调用 Python 服务获取价格预测
func (c *Client) GetPriceForecast(req ForecastRequest) (*ForecastResponse, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/forecast", c.BaseURL)
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bad status: %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	var forecastResp ForecastResponse
	if err := json.NewDecoder(resp.Body).Decode(&forecastResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &forecastResp, nil
}

// ======================= 【高亮-2026-03-18】新增调用 Python 服务回传交易数据 =======================
func (c *Client) RecordTrade(req RecordTradeRequest) error {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/trade/record", c.BaseURL)
	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %d", resp.StatusCode)
	}
	return nil
}