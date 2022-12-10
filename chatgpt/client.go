package chatgpt

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
)

const (
	apiAddr        = "https://chat.openai.com/api"
	backendAPIAddr = "https://chat.openai.com/backend-api"

	userAgent          = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/107.0.0.0 Safari/537.36"
	cookieSessionToken = "__Secure-next-auth.session-token"

	actionNext                = "next"
	roleUser                  = "user"
	contentTypeText           = "text"
	modelTextDavinci002Render = "text-davinci-002-render"

	dataPrefix      = "data: "
	conversationEOF = "[DONE]"
)

type ChatGPT struct {
	httpClient         *http.Client
	sessionToken       string
	accessToken        string
	accessTokenExpires time.Time
}

func NewChatGPT(apiKey string) *ChatGPT {
	return NewChatGPTWithClient(apiKey, http.DefaultClient)
}

func NewChatGPTWithClient(sessionToken string, httpClient *http.Client) *ChatGPT {
	return &ChatGPT{
		sessionToken: sessionToken,
		httpClient:   httpClient,
	}
}

func (c *ChatGPT) NewConversation(conversationId string) *Conversation {
	return &Conversation{
		ChatGPT:         c,
		ConversationId:  conversationId,
		ParentMessageId: uuid.NewString(),
	}
}

func (c *ChatGPT) refreshAccessTokenIfExpired(ctx context.Context) error {
	if c.accessToken == "" || time.Now().After(c.accessTokenExpires) {
		url, _ := url.JoinPath(apiAddr, "auth", "session")

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}

		req.Header.Set("User-Agent", userAgent)
		req.AddCookie(&http.Cookie{
			Name:  cookieSessionToken,
			Value: c.sessionToken,
		})

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
		}

		var authResponse AuthSessionResponse
		err = json.NewDecoder(resp.Body).Decode(&authResponse)
		if err != nil {
			return err
		}

		c.accessToken = authResponse.AccessToken
		c.accessTokenExpires = authResponse.Expires
	}

	return nil
}

type Conversation struct {
	ChatGPT         *ChatGPT
	ConversationId  string
	ParentMessageId string
}

func (c *Conversation) SendMessage(ctx context.Context, message string) (string, error) {
	if err := c.ChatGPT.refreshAccessTokenIfExpired(ctx); err != nil {
		return "", err
	}

	request := &ConversationRequest{
		Action: actionNext,
		Messages: []Message{
			{
				ID:   uuid.NewString(),
				Role: roleUser,
				Content: Content{
					ContentType: contentTypeText,
					Parts:       []string{message},
				},
			},
		},
		ConversationID:  c.ConversationId,
		ParentMessageID: c.ParentMessageId,
		Model:           modelTextDavinci002Render,
	}

	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(request)
	if err != nil {
		return "", err
	}

	url, _ := url.JoinPath(backendAPIAddr, "conversation")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.ChatGPT.accessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.ChatGPT.httpClient.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	respMessage := []byte{}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Bytes()

		if len(line) == 0 {
			continue
		}

		if bytes.HasPrefix(line, []byte(dataPrefix)) {
			data := bytes.TrimPrefix(line, []byte(dataPrefix))

			if bytes.Equal(data, []byte(conversationEOF)) {
				break
			}

			respMessage = data
		}
	}

	var cr ConversationResponse
	if err := json.Unmarshal(respMessage, &cr); err != nil {
		return "", err
	}

	c.ConversationId = cr.ConversationID
	c.ParentMessageId = cr.Message.ID

	return cr.Message.Content.Parts[0], nil
}

type AuthSessionResponse struct {
	Expires     time.Time `json:"expires"`
	AccessToken string    `json:"accessToken"`
}

type ConversationRequest struct {
	Action          string    `json:"action,omitempty"`
	Messages        []Message `json:"messages,omitempty"`
	ConversationID  string    `json:"conversation_id,omitempty"`
	ParentMessageID string    `json:"parent_message_id,omitempty"`
	Model           string    `json:"model,omitempty"`
}

type Message struct {
	ID      string  `json:"id,omitempty"`
	Role    string  `json:"role,omitempty"`
	Content Content `json:"content,omitempty"`
}

type Content struct {
	ContentType string   `json:"content_type,omitempty"`
	Parts       []string `json:"parts,omitempty"`
}

type ConversationResponse struct {
	Message        Message `json:"message"`
	ConversationID string  `json:"conversation_id"`
	Error          string  `json:"error"`
}
