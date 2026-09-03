package metrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	ecmHTTP "github.com/rancher/ecm-distro-tools/http"
)

const slackAPIBase = "https://slack.com/api"

// slackClient talks to the Slack Web API using a bot token. It replaces the
// old incoming-webhook flow, since posting a file + tagging a usergroup
// requires files:write / chat:write scopes that webhooks don't support.
type slackClient struct {
	token  string
	client http.Client
}

func newSlackClient(botToken string) *slackClient {
	return &slackClient{
		token:  botToken,
		client: ecmHTTP.NewClient(time.Second * 60),
	}
}

// uploadReport runs the 3-step Slack external file upload flow and posts a
// message to channelID tagging usergroupID, with the PDF attached.
//
//  1. files.getUploadURLExternal - reserve an upload slot, get a URL + file ID.
//  2. POST the raw file bytes to that URL.
//  3. files.completeUploadExternal - finalize the upload and share it in
//     channelID, with an accompanying message.
func (s *slackClient) uploadReport(filename string, pdf []byte, channelID, usergroupID, summary string) error {
	uploadURL, fileID, err := s.getUploadURL(filename, len(pdf))
	if err != nil {
		return fmt.Errorf("failed to get upload url: %w", err)
	}

	if err := s.putFile(uploadURL, pdf); err != nil {
		return fmt.Errorf("failed to upload file bytes: %w", err)
	}

	initialComment := summary
	if usergroupID != "" {
		initialComment = fmt.Sprintf("<!subteam^%s> %s", usergroupID, summary)
	}

	if err := s.completeUpload(fileID, filename, channelID, initialComment); err != nil {
		return fmt.Errorf("failed to complete upload: %w", err)
	}

	return nil
}

type slackAPIError struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

func (s *slackClient) call(req *http.Request, out interface{}) error {
	req.Header.Set("Authorization", "Bearer "+s.token)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to slack: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("slack API rejected the request with status %d: %s", resp.StatusCode, string(body))
	}

	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("failed to unmarshal slack response: %w", err)
	}

	return nil
}

func (s *slackClient) getUploadURL(filename string, length int) (string, string, error) {
	form := url.Values{}
	form.Set("filename", filename)
	form.Set("length", strconv.Itoa(length))

	req, err := http.NewRequest(http.MethodPost, slackAPIBase+"/files.getUploadURLExternal", bytes.NewBufferString(form.Encode()))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var out struct {
		slackAPIError
		UploadURL string `json:"upload_url"`
		FileID    string `json:"file_id"`
	}
	if err := s.call(req, &out); err != nil {
		return "", "", err
	}
	if !out.OK {
		return "", "", fmt.Errorf("slack error: %s", out.Error)
	}

	return out.UploadURL, out.FileID, nil
}

func (s *slackClient) putFile(uploadURL string, pdf []byte) error {
	req, err := http.NewRequest(http.MethodPost, uploadURL, bytes.NewReader(pdf))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	// this endpoint doesn't require the bot token and doesn't return JSON,
	// just an HTTP 200 on success, so it bypasses the shared call() helper.
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to slack: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("slack upload rejected with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (s *slackClient) completeUpload(fileID, title, channelID, initialComment string) error {
	payload := map[string]interface{}{
		"files": []map[string]string{
			{"id": fileID, "title": title},
		},
		"channel_id":      channelID,
		"initial_comment": initialComment,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, slackAPIBase+"/files.completeUploadExternal", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	var out slackAPIError
	if err := s.call(req, &out); err != nil {
		return err
	}
	if !out.OK {
		return fmt.Errorf("slack error: %s", out.Error)
	}

	return nil
}
