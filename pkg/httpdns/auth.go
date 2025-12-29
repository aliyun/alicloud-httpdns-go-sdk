package httpdns

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
)

// generateSignature 生成签名算法实现
func generateSignature(secretKey, host, timestamp string) string {
	// 构造待签名字符串: host-secret-timestamp
	signString := host + "-" + secretKey + "-" + timestamp

	// 使用MD5算法生成签名
	h := md5.New()
	h.Write([]byte(signString))
	signature := hex.EncodeToString(h.Sum(nil))

	return signature
}

// generateBatchSignature 生成批量解析签名算法
func generateBatchSignature(secretKey string, hosts []string, timestamp string) string {
	// 构造待签名字符串: host1,host2,host3-secret-timestamp
	// 注意：批量解析时host参数就是逗号分隔的域名列表，不需要排序
	hostString := strings.Join(hosts, ",")
	signString := hostString + "-" + secretKey + "-" + timestamp

	// 使用MD5算法生成签名
	h := md5.New()
	h.Write([]byte(signString))
	signature := hex.EncodeToString(h.Sum(nil))

	return signature
}
