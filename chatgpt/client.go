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
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	apiAddr        = "https://chat.openai.com/api"
	backendAPIAddr = "https://chat.openai.com/backend-api"

	userAgent          = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/107.0.0.0 Safari/537.36"
	cookieSessionToken = "__Secure-next-auth.session-token"
	cookieCfClearance  = "cf_clearance"

	actionNext                = "next"
	roleUser                  = "user"
	contentTypeText           = "text"
	modelTextDavinci002Render = "text-davinci-002-render"

	dataPrefix      = "data: "
	conversationEOF = "[DONE]"
)

type ChatGPT struct {
	httpClient         *http.Client
	email              string
	password           string
	sessionToken       string
	userAgent          string
	cfClearance        string
	accessToken        string
	accessTokenExpires time.Time
}

func NewChatGPT(email, password, sessionToken, userAgent, cfClearance string) *ChatGPT {
	return NewChatGPTWithClient(
		email,
		password,
		sessionToken,
		userAgent,
		cfClearance,
		&http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
		})
}

func NewChatGPTWithClient(email, password, sessionToken, userAgent, cfClearance string, httpClient *http.Client) *ChatGPT {
	return &ChatGPT{
		email:        email,
		password:     password,
		sessionToken: sessionToken,
		userAgent:    userAgent,
		cfClearance:  cfClearance,
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
		//if c.email != "" && c.password != "" {
		if c.sessionToken == "" {
			return c.refreshAccessTokenByPassword(ctx)
		} else {
			return c.refreshAccessTokenBySessionToken(ctx)
		}
	}

	return nil
}

func (c *ChatGPT) refreshAccessTokenByPassword(ctx context.Context) error {
	auth, err := NewAuthClient(c.email, c.password, "", nil)
	if err != nil {
		return err
	}

	captcha, err := auth.Begin()
	if err != nil {
		return err
	}

	var answer string
	if captcha.Available() {
		if err := captcha.ToFile("captcha.png"); err != nil {
			return err
		}

		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Captcha answer: ")
		answer, err = reader.ReadString('\n')
		if err != nil {
			return err
		}
		answer = strings.Replace(answer, "\n", "", -1)
	}

	creds, err := auth.Finish(answer)
	if err != nil {
		return err
	}

	c.accessToken = creds.AccessToken
	c.accessTokenExpires = creds.ExpiresAt

	return nil
}

func (c *ChatGPT) refreshAccessTokenBySessionToken(ctx context.Context) error {
	url, _ := url.JoinPath(apiAddr, "auth", "session")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	} else {
		req.Header.Set("User-Agent", userAgent)
	}
	req.AddCookie(&http.Cookie{
		Name:  cookieSessionToken,
		Value: c.sessionToken,
	})
	if c.cfClearance != "" {
		req.AddCookie(&http.Cookie{
			Name:  cookieCfClearance,
			Value: c.cfClearance,
		})
	}

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

	if c.ChatGPT.userAgent != "" {
		req.Header.Set("User-Agent", c.ChatGPT.userAgent)
	} else {
		req.Header.Set("User-Agent", userAgent)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.ChatGPT.accessToken))
	req.Header.Set("Content-Type", "application/json")
	if c.ChatGPT.cfClearance != "" {
		req.AddCookie(&http.Cookie{
			Name:  cookieCfClearance,
			Value: c.ChatGPT.cfClearance,
		})
	}

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

			respMessage = make([]byte, len(data))
			copy(respMessage, data)
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
