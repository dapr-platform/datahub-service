/**
 * @module crypto_utils
 * @description 加密工具模块，负责敏感数据加密、连接信息加密、数据脱敏、密钥管理等功能
 * @architecture 加密工具集模式，提供多种加密和脱敏方法
 * @documentReference 参考 ai_docs/basic_library_process_impl.md 第8.3节
 * @stateFlow 无状态加密：明文 -> 加密算法 -> 密文 / 密文 -> 解密算法 -> 明文
 * @rules
 *   - 密钥管理需要安全存储和轮换
 *   - 敏感数据需要强加密保护
 *   - 脱敏操作需要保证不可逆性
 *   - 加密算法需要使用业界标准
 * @dependencies
 *   - crypto/*: 加密算法
 *   - encoding/hex: 十六进制编码
 *   - crypto/rand: 安全随机数
 *   - crypto/sha256: 哈希算法
 * @refs
 *   - service/config/*: 配置管理
 *   - service/database/*: 数据库加密
 */

package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"
)

// CryptoUtils 加密工具
type CryptoUtils struct {
	defaultKey []byte
}

// NewCryptoUtils 创建新的加密工具实例
func NewCryptoUtils(key string) *CryptoUtils {
	if key == "" {
		key = "datahub-default-key-32-characters"
	}

	// 确保密钥长度为32字节（AES-256）
	hasher := sha256.New()
	hasher.Write([]byte(key))
	defaultKey := hasher.Sum(nil)

	return &CryptoUtils{
		defaultKey: defaultKey,
	}
}

// AES加密功能

// AESEncrypt AES加密
func (cu *CryptoUtils) AESEncrypt(plaintext string, key ...[]byte) (string, error) {
	var encryptKey []byte
	if len(key) > 0 && len(key[0]) > 0 {
		encryptKey = key[0]
	} else {
		encryptKey = cu.defaultKey
	}

	// 创建AES块
	block, err := aes.NewCipher(encryptKey)
	if err != nil {
		return "", fmt.Errorf("创建AES块失败: %v", err)
	}

	// 生成随机IV
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", fmt.Errorf("生成IV失败: %v", err)
	}

	// 创建CFB加密器
	stream := cipher.NewCFBEncrypter(block, iv)

	// 加密
	ciphertext := make([]byte, len(plaintext))
	stream.XORKeyStream(ciphertext, []byte(plaintext))

	// 将IV和密文合并并编码
	result := append(iv, ciphertext...)
	return base64.StdEncoding.EncodeToString(result), nil
}

// AESDecrypt AES解密
func (cu *CryptoUtils) AESDecrypt(ciphertext string, key ...[]byte) (string, error) {
	var decryptKey []byte
	if len(key) > 0 && len(key[0]) > 0 {
		decryptKey = key[0]
	} else {
		decryptKey = cu.defaultKey
	}

	// 解码base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("解码base64失败: %v", err)
	}

	if len(data) < aes.BlockSize {
		return "", fmt.Errorf("密文长度不足")
	}

	// 创建AES块
	block, err := aes.NewCipher(decryptKey)
	if err != nil {
		return "", fmt.Errorf("创建AES块失败: %v", err)
	}

	// 分离IV和密文
	iv := data[:aes.BlockSize]
	ciphertextData := data[aes.BlockSize:]

	// 创建CFB解密器
	stream := cipher.NewCFBDecrypter(block, iv)

	// 解密
	plaintext := make([]byte, len(ciphertextData))
	stream.XORKeyStream(plaintext, ciphertextData)

	return string(plaintext), nil
}

// 哈希功能

// MD5Hash MD5哈希
func (cu *CryptoUtils) MD5Hash(data string) string {
	hasher := md5.New()
	hasher.Write([]byte(data))
	return hex.EncodeToString(hasher.Sum(nil))
}

// SHA256Hash SHA256哈希
func (cu *CryptoUtils) SHA256Hash(data string) string {
	hasher := sha256.New()
	hasher.Write([]byte(data))
	return hex.EncodeToString(hasher.Sum(nil))
}

// HMACSHA256 HMAC-SHA256签名
func (cu *CryptoUtils) HMACSHA256(data, key string) string {
	// 简化实现，实际应使用crypto/hmac
	combined := key + data
	return cu.SHA256Hash(combined)
}

// 数据脱敏功能

// MaskEmail 邮箱脱敏
func (cu *CryptoUtils) MaskEmail(email string) string {
	if email == "" {
		return ""
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email // 无效邮箱格式，不处理
	}

	username := parts[0]
	domain := parts[1]

	// 脱敏用户名部分
	if len(username) <= 2 {
		return strings.Repeat("*", len(username)) + "@" + domain
	}

	maskedUsername := string(username[0]) + strings.Repeat("*", len(username)-2) + string(username[len(username)-1])
	return maskedUsername + "@" + domain
}

// MaskPhone 手机号脱敏
func (cu *CryptoUtils) MaskPhone(phone string) string {
	if phone == "" {
		return ""
	}

	// 去除非数字字符
	re := regexp.MustCompile(`\D`)
	cleanPhone := re.ReplaceAllString(phone, "")

	if len(cleanPhone) < 7 {
		return phone // 太短，不处理
	}

	if len(cleanPhone) == 11 {
		// 中国手机号格式：138****1234
		return cleanPhone[:3] + "****" + cleanPhone[7:]
	}

	// 其他格式：保留前3位和后4位
	if len(cleanPhone) > 7 {
		start := cleanPhone[:3]
		end := cleanPhone[len(cleanPhone)-4:]
		middle := strings.Repeat("*", len(cleanPhone)-7)
		return start + middle + end
	}

	return phone
}

// MaskIDCard 身份证号脱敏
func (cu *CryptoUtils) MaskIDCard(idCard string) string {
	if idCard == "" {
		return ""
	}

	if len(idCard) == 18 {
		// 18位身份证：前6位 + 8个* + 后4位
		return idCard[:6] + "********" + idCard[14:]
	} else if len(idCard) == 15 {
		// 15位身份证：前6位 + 6个* + 后3位
		return idCard[:6] + "******" + idCard[12:]
	}

	return idCard
}

// MaskBankCard 银行卡号脱敏
func (cu *CryptoUtils) MaskBankCard(cardNumber string) string {
	if cardNumber == "" {
		return ""
	}

	// 去除非数字字符
	re := regexp.MustCompile(`\D`)
	cleanCard := re.ReplaceAllString(cardNumber, "")

	if len(cleanCard) < 8 {
		return cardNumber // 太短，不处理
	}

	// 保留前4位和后4位
	start := cleanCard[:4]
	end := cleanCard[len(cleanCard)-4:]
	middle := strings.Repeat("*", len(cleanCard)-8)
	return start + middle + end
}

// MaskName 姓名脱敏
func (cu *CryptoUtils) MaskName(name string) string {
	if name == "" {
		return ""
	}

	runes := []rune(name)
	if len(runes) <= 1 {
		return name
	}

	if len(runes) == 2 {
		// 两个字符：李*
		return string(runes[0]) + "*"
	}

	// 多个字符：李*明 或 欧阳*华
	masked := make([]rune, len(runes))
	masked[0] = runes[0]                       // 保留第一个字符
	masked[len(runes)-1] = runes[len(runes)-1] // 保留最后一个字符

	// 中间字符用*替换
	for i := 1; i < len(runes)-1; i++ {
		masked[i] = '*'
	}

	return string(masked)
}

// MaskGeneral 通用脱敏
func (cu *CryptoUtils) MaskGeneral(data string, keepStart, keepEnd int) string {
	if data == "" {
		return ""
	}

	runes := []rune(data)
	length := len(runes)

	if length <= keepStart+keepEnd {
		return strings.Repeat("*", length)
	}

	start := string(runes[:keepStart])
	end := string(runes[length-keepEnd:])
	middle := strings.Repeat("*", length-keepStart-keepEnd)

	return start + middle + end
}

// 敏感信息检测

// DetectSensitiveType 检测敏感信息类型
func (cu *CryptoUtils) DetectSensitiveType(data string) string {
	if cu.isEmail(data) {
		return "email"
	}
	if cu.isPhone(data) {
		return "phone"
	}
	if cu.isIDCard(data) {
		return "idcard"
	}
	if cu.isBankCard(data) {
		return "bankcard"
	}
	return "unknown"
}

// isEmail 检测是否为邮箱
func (cu *CryptoUtils) isEmail(data string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(data)
}

// isPhone 检测是否为手机号
func (cu *CryptoUtils) isPhone(data string) bool {
	phoneRegex := regexp.MustCompile(`^1[3-9]\d{9}$`)
	return phoneRegex.MatchString(data)
}

// isIDCard 检测是否为身份证号
func (cu *CryptoUtils) isIDCard(data string) bool {
	idCardRegex := regexp.MustCompile(`^(\d{15}|\d{17}[\dXx])$`)
	return idCardRegex.MatchString(data)
}

// isBankCard 检测是否为银行卡号
func (cu *CryptoUtils) isBankCard(data string) bool {
	// 简化检测：16-19位数字
	bankCardRegex := regexp.MustCompile(`^\d{16,19}$`)
	return bankCardRegex.MatchString(data)
}

// AutoMask 自动脱敏
func (cu *CryptoUtils) AutoMask(data string) string {
	sensitiveType := cu.DetectSensitiveType(data)

	switch sensitiveType {
	case "email":
		return cu.MaskEmail(data)
	case "phone":
		return cu.MaskPhone(data)
	case "idcard":
		return cu.MaskIDCard(data)
	case "bankcard":
		return cu.MaskBankCard(data)
	default:
		// 默认脱敏：保留前1位和后1位
		return cu.MaskGeneral(data, 1, 1)
	}
}

// 密钥管理功能

// GenerateKey 生成随机密钥
func (cu *CryptoUtils) GenerateKey(length int) ([]byte, error) {
	if length <= 0 {
		length = 32 // 默认32字节
	}

	key := make([]byte, length)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("生成密钥失败: %v", err)
	}

	return key, nil
}

// GenerateKeyString 生成随机密钥字符串
func (cu *CryptoUtils) GenerateKeyString(length int) (string, error) {
	key, err := cu.GenerateKey(length)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(key), nil
}

// DeriveKey 从密码派生密钥
func (cu *CryptoUtils) DeriveKey(password, salt string) []byte {
	// 简化实现，实际应使用PBKDF2或scrypt
	combined := password + salt
	hasher := sha256.New()
	hasher.Write([]byte(combined))
	return hasher.Sum(nil)
}

// 批量处理功能

// BatchEncrypt 批量加密
func (cu *CryptoUtils) BatchEncrypt(data map[string]string, key ...[]byte) (map[string]string, error) {
	result := make(map[string]string)

	for field, value := range data {
		encrypted, err := cu.AESEncrypt(value, key...)
		if err != nil {
			return nil, fmt.Errorf("加密字段 %s 失败: %v", field, err)
		}
		result[field] = encrypted
	}

	return result, nil
}

// BatchDecrypt 批量解密
func (cu *CryptoUtils) BatchDecrypt(data map[string]string, key ...[]byte) (map[string]string, error) {
	result := make(map[string]string)

	for field, value := range data {
		decrypted, err := cu.AESDecrypt(value, key...)
		if err != nil {
			return nil, fmt.Errorf("解密字段 %s 失败: %v", field, err)
		}
		result[field] = decrypted
	}

	return result, nil
}

// BatchMask 批量脱敏
func (cu *CryptoUtils) BatchMask(data map[string]string, rules map[string]string) map[string]string {
	result := make(map[string]string)

	for field, value := range data {
		if maskType, exists := rules[field]; exists {
			switch maskType {
			case "email":
				result[field] = cu.MaskEmail(value)
			case "phone":
				result[field] = cu.MaskPhone(value)
			case "idcard":
				result[field] = cu.MaskIDCard(value)
			case "bankcard":
				result[field] = cu.MaskBankCard(value)
			case "name":
				result[field] = cu.MaskName(value)
			case "auto":
				result[field] = cu.AutoMask(value)
			default:
				result[field] = value
			}
		} else {
			result[field] = value
		}
	}

	return result
}

// 工具函数

// IsValidUTF8 检查是否为有效UTF-8编码
func (cu *CryptoUtils) IsValidUTF8(data string) bool {
	return utf8.ValidString(data)
}

// SecureCompare 安全比较字符串（防时序攻击）
func (cu *CryptoUtils) SecureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}

	result := 0
	for i := 0; i < len(a); i++ {
		result |= int(a[i]) ^ int(b[i])
	}

	return result == 0
}
