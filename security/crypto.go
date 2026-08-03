// Package security 提供 API 密钥等敏感数据的静态加密能力。
// 使用 AES-256-GCM，密文格式：enc:v1:<base64(nonce||ciphertext)>。
// 加密密钥从配置 security.encrypt_key 读取；未配置时回退到 JWT Secret 派生，
// 并在启动时记录警告（生产环境必须显式配置）。
package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/chihqiang/infra-go/hash"
	"github.com/chihqiang/infra-go/logger"
)

const (
	encPrefix    = "enc:v1:"
	gcmOverhead  = 16
	versionNonce = 12 // GCM 标准 nonce 长度
)

// Cipher 静态数据加密器。
type Cipher struct {
	aead cipher.AEAD
}

// keyFromConfig 从 32 字节 hex 字符串构造密钥；非法时返回错误。
func keyFromConfig(hexKey string) ([]byte, error) {
	if len(hexKey) == 64 {
		key := make([]byte, 32)
		if _, err := fmt.Sscanf(hexKey, "%64x", &key); err == nil {
			return key, nil
		}
	}
	return nil, errors.New("security.encrypt_key 必须是 64 位十六进制（32 字节）")
}

// New 创建加密器。encryptKey 为空时使用 fallbackKey 派生密钥并告警。
func New(encryptKey, fallbackKey string) (*Cipher, error) {
	var key []byte
	var err error
	if encryptKey != "" {
		key, err = keyFromConfig(encryptKey)
		if err != nil {
			return nil, err
		}
	} else {
		sum := sha256.Sum256([]byte(fallbackKey))
		key = sum[:]
		logger.Warn("security: 未配置 security.encrypt_key，使用 JWT Secret 派生的密钥（生产环境请显式配置）")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead}, nil
}

// Encrypt 加密字符串，返回带版本前缀的密文。空字符串直接返回空。
func (c *Cipher) Encrypt(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	nonce := make([]byte, versionNonce)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := c.aead.Seal(nonce, nonce, []byte(plain), nil)
	return encPrefix + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密带版本前缀的密文。空字符串直接返回空。
// 无法识别前缀时按原始值返回（兼容历史明文数据）。
func (c *Cipher) Decrypt(encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}
	if !strings.HasPrefix(encrypted, encPrefix) {
		return encrypted, nil
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(encrypted, encPrefix))
	if err != nil {
		return "", fmt.Errorf("security: base64 decode failed: %w", err)
	}
	if len(raw) < versionNonce+gcmOverhead {
		return "", errors.New("security: ciphertext too short")
	}
	nonce := raw[:versionNonce]
	body := raw[versionNonce:]
	plain, err := c.aead.Open(nil, nonce, body, nil)
	if err != nil {
		return "", fmt.Errorf("security: decrypt failed: %w", err)
	}
	return string(plain), nil
}

// IsEncrypted 判断字符串是否为加密格式。
func IsEncrypted(s string) bool {
	return strings.HasPrefix(s, encPrefix)
}

// SHA256Hex 返回字符串的 SHA-256 十六进制摘要，用于密钥查询索引。
func SHA256Hex(s string) string {
	return hash.SHA256String(s)
}
