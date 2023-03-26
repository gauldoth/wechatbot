package gtp

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/869413421/wechatbot/config"
)

const BASEURL = "https://api.openai.com/v1/"

// ChatGPTResponseBody 请求体
type ChatGPTResponseBody struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int                    `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChoiceItem           `json:"choices"`
	Usage   map[string]interface{} `json:"usage"`
	Error   map[string]interface{} `json:"error"`
}

type ChoiceItem struct {
	Index        int            `json:"index"`
	FinishReason string         `json:"finish_reason"`
	Message      ChatGPTMessage `json:"message"`
}

// ChatGPTRequestBody 响应体
type ChatGPTRequestBody struct {
	Model            string           `json:"model"`
	Messages         []ChatGPTMessage `json:"messages"`
	MaxTokens        int              `json:"max_tokens"`
	Temperature      float32          `json:"temperature"`
	TopP             int              `json:"top_p"`
	FrequencyPenalty int              `json:"frequency_penalty"`
	PresencePenalty  int              `json:"presence_penalty"`
	Stop             []string         `json:"stop"`
	User             string           `json:"user"`
}

type ChatGPTMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

var conversations map[string][]ChatGPTMessage

// Completions gtp文本模型回复
// curl https://api.openai.com/v1/chat/completions
// -H "Content-Type: application/json"
// -H "Authorization: Bearer your chatGPT key"
// -d '{"model": "gpt-3.5-turbo", "messages": [{"role":"system", "content":"You are a assistant"}, {"role":"user", "content": "give me good song"}], "temperature": 0, "max_tokens": 7}'
func Completions(msg string, conversationKey string) (string, error) {
	if _, ok := conversations[conversationKey]; !ok {
		conversation := make([]ChatGPTMessage, 0)
		conversation = append(conversation, ChatGPTMessage{Role: "system", Content: "You are a helpful assistant."})
		conversations[conversationKey] = conversation
	}

	if strings.Contains(msg, "gpt:reset") {
		conversation := make([]ChatGPTMessage, 0)
		conversation = append(conversation, ChatGPTMessage{Role: "system", Content: "You are a helpful assistant."})
		conversations[conversationKey] = conversation
		return "conversation initialized", nil
	}

	conversations[conversationKey] = append(conversations[conversationKey], ChatGPTMessage{Role: "user", Content: msg})
	requestBody := ChatGPTRequestBody{
		Model: "gpt-3.5-turbo",
		Messages: conversations[conversationKey],
		MaxTokens:        2048,
		Temperature:      0.7,
		TopP:             1,
		FrequencyPenalty: 0,
		PresencePenalty:  0,
	}
	requestData, err := json.Marshal(requestBody)

	if err != nil {
		return "", err
	}
	log.Printf("request gtp json string : %v", string(requestData))
	req, err := http.NewRequest("POST", BASEURL+"chat/completions", bytes.NewBuffer(requestData))
	if err != nil {
		return "", err
	}

	apiKey := config.LoadConfig().ApiKey
	proxy := config.LoadConfig().Proxy
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	var client *http.Client
	if len(proxy) == 0 {
		client = &http.Client{}
	} else {
		proxyAddr, _ := url.Parse(proxy)
		client = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyAddr),
			},
		}
	}

	response, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	gptResponseBody := &ChatGPTResponseBody{}
	log.Println(string(body))
	err = json.Unmarshal(body, gptResponseBody)
	if err != nil {
		return "", err
	}
	var reply string
	if len(gptResponseBody.Choices) > 0 {
		for _, v := range gptResponseBody.Choices {
			reply = v.Message.Content
			conversations[conversationKey] = append(conversations[conversationKey], ChatGPTMessage{Role: "assistant", Content: reply})
			break
		}
	} else {
		return string(body), nil
	}
	log.Printf("gpt response text: %s \n", reply)
	return reply, nil
}


func init() {
	conversations = make(map[string][]ChatGPTMessage)
}