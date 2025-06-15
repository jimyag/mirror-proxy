package execute

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTargetURL(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantURL string
		wantErr bool
	}{
		{
			name:    "完整的 HTTPS URL",
			path:    "/https://example.com/path",
			wantURL: "https://example.com/path",
			wantErr: false,
		},
		{
			name:    "单斜杠 HTTPS URL",
			path:    "/https:/example.com/path",
			wantURL: "https://example.com/path",
			wantErr: false,
		},
		{
			name:    "完整 HTTP URL",
			path:    "/http://example.com/path",
			wantURL: "http://example.com/path",
			wantErr: false,
		},
		{
			name:    "无协议 URL",
			path:    "/example.com/path",
			wantURL: "https://example.com/path",
			wantErr: false,
		},
		{
			name:    "带查询参数的 URL",
			path:    "/example.com/path?param=value",
			wantURL: "https://example.com/path?param=value",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)

			got, err := getTargetURL(req)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.wantURL, got.String())
		})
	}
}
