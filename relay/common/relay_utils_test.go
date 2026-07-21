package common

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeURLForLogMasksSensitiveQueryValues(t *testing.T) {
	rawURL := "https://example.test/v1beta/models/gemini:streamGenerateContent?alt=sse&key=sk-secret&access_token=ya29-secret&api-version=2024-02-01"

	got := SanitizeURLForLog(rawURL)

	assert.NotContains(t, got, "sk-secret")
	assert.NotContains(t, got, "ya29-secret")
	parsedURL, err := url.Parse(got)
	require.NoError(t, err)
	query := parsedURL.Query()
	assert.Equal(t, "***masked***", query.Get("key"))
	assert.Equal(t, "***masked***", query.Get("access_token"))
	assert.Equal(t, "sse", query.Get("alt"))
	assert.Equal(t, "2024-02-01", query.Get("api-version"))
}

func TestSanitizeURLForLogMasksAWSAndSecretLikeQueryKeys(t *testing.T) {
	rawURL := "https://example.test/path?X-Amz-Credential=credential&X-Amz-Signature=signature&session_token=session&client_secret=secret&model=gpt-test"

	got := SanitizeURLForLog(rawURL)

	assert.NotContains(t, got, "X-Amz-Credential=credential")
	assert.NotContains(t, got, "X-Amz-Signature=signature")
	assert.NotContains(t, got, "session_token=session")
	assert.NotContains(t, got, "client_secret=secret")
	parsedURL, err := url.Parse(got)
	require.NoError(t, err)
	query := parsedURL.Query()
	assert.Equal(t, "***masked***", query.Get("X-Amz-Credential"))
	assert.Equal(t, "***masked***", query.Get("X-Amz-Signature"))
	assert.Equal(t, "***masked***", query.Get("session_token"))
	assert.Equal(t, "***masked***", query.Get("client_secret"))
	assert.Equal(t, "gpt-test", query.Get("model"))
}

func TestSanitizeURLForLogKeepsURLWithoutSensitiveQuery(t *testing.T) {
	rawURL := "https://example.test/v1/chat/completions?api-version=2024-02-01&alt=sse"

	got := SanitizeURLForLog(rawURL)

	assert.Equal(t, rawURL, got)
}

// TestGetFullRequestURLOpenAICompatibleStripsV1Prefix locks in the one-api
// compatible behavior: when a user picks the OpenAI Compatible channel type
// and fills in a non-standard root (e.g. the VolcEngine Agent Plan endpoint),
// the leading "/v1" segment of the OpenAI request path must be stripped so
// the final URL is correct instead of double-prefixed.
func TestGetFullRequestURLOpenAICompatibleStripsV1Prefix(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		requestURL  string
		channelType int
		want        string
	}{
		{
			name:        "agent plan root strips /v1/chat/completions",
			baseURL:     "https://ark.cn-beijing.volces.com/api/plan/v3",
			requestURL:  "/v1/chat/completions",
			channelType: constant.ChannelTypeOpenAICompatible,
			want:        "https://ark.cn-beijing.volces.com/api/plan/v3/chat/completions",
		},
		{
			name:        "trailing slash on baseURL is trimmed",
			baseURL:     "https://ark.cn-beijing.volces.com/api/plan/v3/",
			requestURL:  "/v1/chat/completions",
			channelType: constant.ChannelTypeOpenAICompatible,
			want:        "https://ark.cn-beijing.volces.com/api/plan/v3/chat/completions",
		},
		{
			name:        "zhipu coding paas v4 root strips /v1",
			baseURL:     "https://open.bigmodel.cn/api/coding/paas/v4",
			requestURL:  "/v1/chat/completions",
			channelType: constant.ChannelTypeOpenAICompatible,
			want:        "https://open.bigmodel.cn/api/coding/paas/v4/chat/completions",
		},
		{
			name:        "standard OpenAI root still works",
			baseURL:     "https://api.openai.com",
			requestURL:  "/v1/chat/completions",
			channelType: constant.ChannelTypeOpenAICompatible,
			want:        "https://api.openai.com/chat/completions",
		},
		{
			name:        "embeddings path strips /v1",
			baseURL:     "https://ark.cn-beijing.volces.com/api/plan/v3",
			requestURL:  "/v1/embeddings",
			channelType: constant.ChannelTypeOpenAICompatible,
			want:        "https://ark.cn-beijing.volces.com/api/plan/v3/embeddings",
		},
		{
			name:        "path without /v1 prefix is unchanged",
			baseURL:     "https://example.test/api/v3",
			requestURL:  "/models",
			channelType: constant.ChannelTypeOpenAICompatible,
			want:        "https://example.test/api/v3/models",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetFullRequestURL(tt.baseURL, tt.requestURL, tt.channelType)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestGetFullRequestURLOpenAITypeStillConcatenatesVerbatim guards against
// accidental regression: the plain OpenAI channel type must keep concatenating
// baseURL + requestURL verbatim (it never stripped /v1).
func TestGetFullRequestURLOpenAITypeStillConcatenatesVerbatim(t *testing.T) {
	got := GetFullRequestURL("https://api.openai.com", "/v1/chat/completions", constant.ChannelTypeOpenAI)
	assert.Equal(t, "https://api.openai.com/v1/chat/completions", got)
}

func TestValidateMultipartDirectNormalizesImageField(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := strings.NewReader(`{"model":"wan2.7-i2v","prompt":"animate","image":" https://example.com/first.png "}`)
	request := httptest.NewRequest(http.MethodPost, "/v1/video/generations", body)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request
	info := &RelayInfo{
		TaskRelayInfo: &TaskRelayInfo{},
	}

	taskErr := ValidateMultipartDirect(context, info)

	require.Nil(t, taskErr)
	storedReq, err := GetTaskRequest(context)
	require.NoError(t, err)
	require.Equal(t, []string{"https://example.com/first.png"}, storedReq.Images)
	require.Equal(t, constant.TaskActionGenerate, info.Action)
}

// TestTaskDurationBounds guards the billing invariant that user-supplied
// video duration (a quota multiplier via OtherRatio "seconds") is bounded, so
// it can never overflow quota calculation into a negative charge.
func TestTaskDurationBounds(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newContext := func(t *testing.T, body string) (*gin.Context, *RelayInfo) {
		request := httptest.NewRequest(http.MethodPost, "/v1/video/generations", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		context, _ := gin.CreateTestContext(httptest.NewRecorder())
		context.Request = request
		return context, &RelayInfo{TaskRelayInfo: &TaskRelayInfo{}}
	}

	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name:    "huge duration is rejected",
			body:    `{"model":"sora-2","prompt":"a cat","duration":9999999999}`,
			wantErr: true,
		},
		{
			name:    "huge seconds string is rejected",
			body:    `{"model":"sora-2","prompt":"a cat","seconds":"9999999999"}`,
			wantErr: true,
		},
		{
			name:    "negative duration is rejected",
			body:    `{"model":"sora-2","prompt":"a cat","duration":-8}`,
			wantErr: true,
		},
		{
			name: "normal duration is accepted",
			body: `{"model":"sora-2","prompt":"a cat","seconds":"8"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" (multipart direct)", func(t *testing.T) {
			context, info := newContext(t, tt.body)
			taskErr := ValidateMultipartDirect(context, info)
			if tt.wantErr {
				require.NotNil(t, taskErr)
				require.Equal(t, "invalid_seconds", taskErr.Code)
			} else {
				require.Nil(t, taskErr)
			}
		})
		t.Run(tt.name+" (basic task request)", func(t *testing.T) {
			context, info := newContext(t, tt.body)
			taskErr := ValidateBasicTaskRequest(context, info, constant.TaskActionGenerate)
			if tt.wantErr {
				require.NotNil(t, taskErr)
				require.Equal(t, "invalid_seconds", taskErr.Code)
			} else {
				require.Nil(t, taskErr)
			}
		})
	}
}
