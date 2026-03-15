//go:build !solution

package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

const apiURL = "https://api.telegram.org/bot%s/%s"

// HTTPClient реализует Client через HTTP API Telegram.
type HTTPClient struct {
	token      string
	httpClient *http.Client
}

// NewHTTPClient создаёт нового HTTP клиента Telegram.
func NewHTTPClient(token string) *HTTPClient {
	return &HTTPClient{
		token:      token,
		httpClient: &http.Client{},
	}
}

// SendMessage отправляет сообщение.
func (c *HTTPClient) SendMessage(chatID int64, text string, opts *SendOptions) (*Message, error) {
	params := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	}

	if opts != nil {
		if opts.ParseMode != "" {
			params["parse_mode"] = opts.ParseMode
		}
		if opts.ReplyMarkup != nil {
			params["reply_markup"] = opts.ReplyMarkup
		}
	}

	data, err := c.doRequest("sendMessage", params)

	if err != nil {
		return nil, err
	}

	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}

	return &msg, nil
}

// EditMessage редактирует сообщение.
func (c *HTTPClient) EditMessage(chatID int64, messageID int, text string, opts *SendOptions) error {
	params := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"message_id": messageID,
	}

	if opts != nil {
		if opts.ParseMode != "" {
			params["parse_mode"] = opts.ParseMode
		}
		if opts.ReplyMarkup != nil {
			params["reply_markup"] = opts.ReplyMarkup
		}
	}

	_, err := c.doRequest("editMessageText", params)

	return err
}

// DeleteMessage удаляет сообщение.
func (c *HTTPClient) DeleteMessage(chatID int64, messageID int) error {
	params := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
	}

	_, err := c.doRequest("deleteMessage", params)

	return err
}

// AnswerCallback отвечает на callback query.
func (c *HTTPClient) AnswerCallback(callbackID string, text string) error {
	params := map[string]interface{}{
		"callback_query_id": callbackID,
		"text":              text,
	}

	_, err := c.doRequest("answerCallbackQuery", params)

	return err
}

// GetUpdates получает обновления.
func (c *HTTPClient) GetUpdates(offset int, timeout int) ([]Update, error) {
	params := map[string]interface{}{
		"offset":  offset,
		"timeout": timeout,
	}

	data, err := c.doRequest("getUpdates", params)

	if err != nil {
		return nil, err
	}

	var updates []Update
	if err := json.Unmarshal(data, &updates); err != nil {
		return nil, err
	}

	return updates, nil
}

// GetFile получает информацию о файле.
func (c *HTTPClient) GetFile(fileID string) (string, error) {
	params := map[string]interface{}{
		"file_id": fileID,
	}

	data, err := c.doRequest("getFile", params)

	if err != nil {
		return "", err
	}

	var file File
	if err := json.Unmarshal(data, &file); err != nil {
		return "", err
	}

	return file.FilePath, nil
}

// DownloadFile скачивает файл.
func (c *HTTPClient) DownloadFile(filePath string) ([]byte, error) {
	url := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", c.token, filePath)

	resp, err := c.httpClient.Get(url)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download file error: code %d", resp.StatusCode)
	}

	bytes, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	return bytes, nil
}

// SendDocument отправляет файл как документ.
func (c *HTTPClient) SendDocument(chatID int64, fileName string, data []byte) error {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("document", fileName)
	if err != nil {
		return err
	}

	if _, err := io.Copy(part, bytes.NewReader(data)); err != nil {
		return err
	}

	if err := writer.WriteField("chat_id", fmt.Sprintf("%d", chatID)); err != nil {
		return err
	}

	if err := writer.Close(); err != nil {
		return err
	}

	url := fmt.Sprintf(apiURL, c.token, "sendDocument")
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("send document error: code %d", resp.StatusCode)
	}

	return nil
}

// doRequest выполняет запрос к Telegram API.
func (c *HTTPClient) doRequest(method string, params map[string]interface{}) (json.RawMessage, error) {
	url := fmt.Sprintf(apiURL, c.token, method)

	body, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result"`
		Error  string          `json:"description"`
	}

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	if !result.OK {
		return nil, fmt.Errorf("telegram api error: %s", result.Error)
	}

	return result.Result, nil
}
