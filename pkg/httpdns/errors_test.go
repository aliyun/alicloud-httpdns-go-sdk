package httpdns

import (
	"errors"
	"testing"
)

func TestHTTPDNSError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *HTTPDNSError
		expected string
	}{
		{
			name: "with domain",
			err: &HTTPDNSError{
				Op:     "resolve",
				Domain: "example.com",
				Err:    errors.New("network error"),
			},
			expected: "httpdns resolve example.com: network error",
		},
		{
			name: "without domain",
			err: &HTTPDNSError{
				Op:  "init",
				Err: errors.New("config error"),
			},
			expected: "httpdns init: config error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("HTTPDNSError.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestHTTPDNSError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	httpDNSErr := &HTTPDNSError{
		Op:  "test",
		Err: originalErr,
	}

	if unwrapped := httpDNSErr.Unwrap(); unwrapped != originalErr {
		t.Errorf("HTTPDNSError.Unwrap() = %v, want %v", unwrapped, originalErr)
	}
}

func TestNewHTTPDNSError(t *testing.T) {
	originalErr := errors.New("test error")
	httpDNSErr := NewHTTPDNSError("resolve", "example.com", originalErr)

	if httpDNSErr.Op != "resolve" {
		t.Errorf("NewHTTPDNSError() Op = %v, want %v", httpDNSErr.Op, "resolve")
	}

	if httpDNSErr.Domain != "example.com" {
		t.Errorf("NewHTTPDNSError() Domain = %v, want %v", httpDNSErr.Domain, "example.com")
	}

	if httpDNSErr.Err != originalErr {
		t.Errorf("NewHTTPDNSError() Err = %v, want %v", httpDNSErr.Err, originalErr)
	}
}

func TestErrorTypes(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "ErrTooManyDomains",
			err:  ErrTooManyDomains,
			want: "too many domains, maximum 5 domains allowed per batch request",
		},
		{
			name: "ErrInvalidDomain",
			err:  ErrInvalidDomain,
			want: "invalid domain name",
		},
		{
			name: "ErrServiceUnavailable",
			err:  ErrServiceUnavailable,
			want: "service unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error message = %v, want %v", got, tt.want)
			}
		})
	}
}
