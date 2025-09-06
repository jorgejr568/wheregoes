package utils

import "testing"

func TestIsUrl(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid http url",
			input:    "http://example.com",
			expected: true,
		},
		{
			name:     "valid https url",
			input:    "https://example.com",
			expected: true,
		},
		{
			name:     "valid http url with path",
			input:    "http://example.com/path",
			expected: true,
		},
		{
			name:     "valid https url with query params",
			input:    "https://example.com/path?query=value",
			expected: true,
		},
		{
			name:     "invalid url without protocol",
			input:    "example.com",
			expected: false,
		},
		{
			name:     "invalid url with wrong protocol",
			input:    "ftp://example.com",
			expected: false,
		},
		{
			name:     "invalid url with typo in protocol",
			input:    "htttp://example.com",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "random string",
			input:    "not a url",
			expected: false,
		},
		{
			name:     "url with port",
			input:    "https://example.com:8080",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsUrl(tt.input)
			if result != tt.expected {
				t.Errorf("IsUrl(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}