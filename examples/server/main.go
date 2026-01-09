package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/aliyun/alicloud-httpdns-go-sdk/pkg/httpdns"
)

// DNSResponse API 响应结构
type DNSResponse struct {
	Domain    string   `json:"domain"`
	ClientIP  string   `json:"client_ip"`
	IPv4      []string `json:"ipv4"`
	IPv6      []string `json:"ipv6"`
	TTL       int      `json:"ttl"`
	Source    string   `json:"source"`
	Timestamp string   `json:"timestamp"`
	Error     string   `json:"error,omitempty"`
}

// DNSServer HTTP DNS 服务器
type DNSServer struct {
	client httpdns.Client
}

// NewDNSServer 创建新的 DNS 服务器
func NewDNSServer() (*DNSServer, error) {
	config := httpdns.DefaultConfig()
	config.AccountID = "your-account-id" // 替换为你的 Account ID
	// config.SecretKey = "your-secret-key" // 可选：启用鉴权

	// 启用所有功能
	config.EnableMetrics = true

	client, err := httpdns.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &DNSServer{client: client}, nil
}

// Close 关闭服务器
func (s *DNSServer) Close() error {
	return s.client.Close()
}

// handleResolve 处理单域名解析请求
func (s *DNSServer) handleResolve(w http.ResponseWriter, r *http.Request) {
	// 设置响应头
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 获取参数
	domain := r.URL.Query().Get("domain")
	clientIP := r.Header.Get("X-Real-IP")
	if clientIP == "" {
		clientIP = r.Header.Get("X-Forwarded-For")
	}
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}

	if domain == "" {
		http.Error(w, "Missing domain parameter", http.StatusBadRequest)
		return
	}

	// 创建上下文
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// 解析域名
	opts := []httpdns.ResolveOption{}
	if clientIP != "" {
		opts = append(opts, httpdns.WithClientIP(clientIP))
	}
	result, err := s.client.Resolve(ctx, domain, opts...)

	response := DNSResponse{
		Domain:    domain,
		ClientIP:  clientIP,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if err != nil {
		response.Error = err.Error()
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		// 转换 IP 地址为字符串
		for _, ip := range result.IPv4 {
			response.IPv4 = append(response.IPv4, ip.String())
		}
		for _, ip := range result.IPv6 {
			response.IPv6 = append(response.IPv6, ip.String())
		}

		response.TTL = int(result.TTL.Seconds())
		response.Source = result.Source.String()
	}

	// 返回 JSON 响应
	json.NewEncoder(w).Encode(response)
}

// handleBatchResolve 处理批量域名解析请求
func (s *DNSServer) handleBatchResolve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析请求体
	var request struct {
		Domains []string `json:"domains"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if len(request.Domains) == 0 {
		http.Error(w, "No domains provided", http.StatusBadRequest)
		return
	}

	// 获取客户端 IP
	clientIP := r.Header.Get("X-Real-IP")
	if clientIP == "" {
		clientIP = r.Header.Get("X-Forwarded-For")
	}
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}

	// 创建上下文
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// 批量解析
	opts := []httpdns.ResolveOption{}
	if clientIP != "" {
		opts = append(opts, httpdns.WithClientIP(clientIP))
	}
	results, err := s.client.ResolveBatch(ctx, request.Domains, opts...)

	var responses []DNSResponse

	if err != nil {
		// 如果批量解析失败，为每个域名创建错误响应
		for _, domain := range request.Domains {
			responses = append(responses, DNSResponse{
				Domain:    domain,
				ClientIP:  clientIP,
				Error:     err.Error(),
				Timestamp: time.Now().Format(time.RFC3339),
			})
		}
	} else {
		// 转换结果
		for _, result := range results {
			response := DNSResponse{
				Domain:    result.Domain,
				ClientIP:  clientIP,
				TTL:       int(result.TTL.Seconds()),
				Source:    result.Source.String(),
				Timestamp: result.Timestamp.Format(time.RFC3339),
			}

			if result.Error != nil {
				response.Error = result.Error.Error()
			} else {
				for _, ip := range result.IPv4 {
					response.IPv4 = append(response.IPv4, ip.String())
				}
				for _, ip := range result.IPv6 {
					response.IPv6 = append(response.IPv6, ip.String())
				}
			}

			responses = append(responses, response)
		}
	}

	// 返回 JSON 响应
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": responses,
	})
}

// handleMetrics 处理指标查询请求
func (s *DNSServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	stats := s.client.GetMetrics()
	json.NewEncoder(w).Encode(stats)
}

// handleHealth 处理健康检查请求
func (s *DNSServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	healthy := s.client.IsHealthy()
	status := "ok"
	if !healthy {
		status = "unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	response := map[string]interface{}{
		"status":      status,
		"timestamp":   time.Now().Format(time.RFC3339),
		"service_ips": s.client.GetServiceIPs(),
	}

	json.NewEncoder(w).Encode(response)
}

func main() {
	// 创建 DNS 服务器
	server, err := NewDNSServer()
	if err != nil {
		log.Fatalf("Failed to create DNS server: %v", err)
	}
	defer server.Close()

	// 设置路由
	http.HandleFunc("/resolve", server.handleResolve)
	http.HandleFunc("/batch", server.handleBatchResolve)
	http.HandleFunc("/metrics", server.handleMetrics)
	http.HandleFunc("/health", server.handleHealth)

	// 静态文件服务（可选）
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <title>HTTPDNS Server</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .endpoint { margin: 20px 0; padding: 10px; background: #f5f5f5; }
        code { background: #e0e0e0; padding: 2px 4px; }
    </style>
</head>
<body>
    <h1>HTTPDNS Server</h1>
    <p>这是一个基于 HTTPDNS Go SDK 的 HTTP DNS 服务器示例。</p>
    
    <div class="endpoint">
        <h3>单域名解析</h3>
        <p><code>GET /resolve?domain=example.com</code></p>
    </div>
    
    <div class="endpoint">
        <h3>批量域名解析</h3>
        <p><code>POST /batch</code></p>
        <p>请求体: <code>{"domains": ["example.com", "google.com"]}</code></p>
    </div>
    
    <div class="endpoint">
        <h3>指标查询</h3>
        <p><code>GET /metrics</code></p>
    </div>
    
    <div class="endpoint">
        <h3>健康检查</h3>
        <p><code>GET /health</code></p>
    </div>
</body>
</html>
			`)
		} else {
			http.NotFound(w, r)
		}
	})

	// 启动服务器
	port := ":8080"
	fmt.Printf("HTTPDNS Server starting on port %s\n", port)
	fmt.Printf("访问 http://localhost%s 查看 API 文档\n", port)

	log.Fatal(http.ListenAndServe(port, nil))
}
