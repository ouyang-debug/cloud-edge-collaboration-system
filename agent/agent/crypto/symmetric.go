package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"

	"github.com/tjfoc/gmsm/sm4"
)

// GenerateAESKey 生成指定长度的AES密钥
func GenerateAESKey(bits int) ([]byte, error) {
	key := make([]byte, bits/8)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// AESKeyToBase64 将AES密钥转换为Base64格式
func AESKeyToBase64(key []byte) string {
	return base64.StdEncoding.EncodeToString(key)
}

// Base64ToAESKey 将Base64格式转换为AES密钥
func Base64ToAESKey(keyBase64 string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(keyBase64)
}

// AESEncrypt 使用AES-GCM模式加密数据
func AESEncrypt(key []byte, plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 创建GCM实例
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 创建随机nonce
	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		return nil, err
	}

	// 加密数据
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// AESDecrypt 使用AES-GCM模式解密数据
func AESDecrypt(key []byte, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 创建GCM实例
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 检查密文长度
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	// 分离nonce和密文
	nonce := ciphertext[:nonceSize]
	ciphertext = ciphertext[nonceSize:]

	// 解密数据
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// AESEncryptBase64 使用AES-GCM模式加密数据并返回Base64编码
func AESEncryptBase64(key []byte, plaintext string) (string, error) {
	ciphertext, err := AESEncrypt(key, []byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// AESDecryptBase64 解密Base64编码的AES-GCM加密数据
func AESDecryptBase64(key []byte, ciphertextBase64 string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", err
	}
	plaintext, err := AESDecrypt(key, ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// ------------ SM4算法 ------------

// GenerateSM4Key 生成SM4密钥（128位，16字节）
func GenerateSM4Key() ([]byte, error) {
	key := make([]byte, sm4.BlockSize)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// SM4KeyToBase64 将SM4密钥转换为Base64格式
func SM4KeyToBase64(key []byte) string {
	return base64.StdEncoding.EncodeToString(key)
}

// Base64ToSM4Key 将Base64格式转换为SM4密钥
func Base64ToSM4Key(keyBase64 string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, err
	}
	if len(key) != sm4.BlockSize {
		return nil, errors.New("invalid SM4 key length, must be 16 bytes")
	}
	return key, nil
}

// SM4EncryptECB 使用SM4-ECB模式加密数据
func SM4EncryptECB(key, plaintext []byte) ([]byte, error) {
	block, err := sm4.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 对明文进行填充
	padding := sm4.BlockSize - len(plaintext)%sm4.BlockSize
	paddedPlaintext := append(plaintext, make([]byte, padding)...)
	for i := len(plaintext); i < len(paddedPlaintext); i++ {
		paddedPlaintext[i] = byte(padding)
	}

	ciphertext := make([]byte, len(paddedPlaintext))
	// ECB模式加密
	for i := 0; i < len(paddedPlaintext); i += sm4.BlockSize {
		block.Encrypt(ciphertext[i:i+sm4.BlockSize], paddedPlaintext[i:i+sm4.BlockSize])
	}

	return ciphertext, nil
}

// SM4DecryptECB 使用SM4-ECB模式解密数据
func SM4DecryptECB(key, ciphertext []byte) ([]byte, error) {
	block, err := sm4.NewCipher(key)
	if err != nil {
		return nil, err
	}

	plaintext := make([]byte, len(ciphertext))
	// ECB模式解密
	for i := 0; i < len(ciphertext); i += sm4.BlockSize {
		block.Decrypt(plaintext[i:i+sm4.BlockSize], ciphertext[i:i+sm4.BlockSize])
	}

	// 移除填充
	padding := int(plaintext[len(plaintext)-1])
	if padding < 1 || padding > sm4.BlockSize {
		return nil, errors.New("invalid padding")
	}

	return plaintext[:len(plaintext)-padding], nil
}

// SM4EncryptCBC 使用SM4-CBC模式加密数据
func SM4EncryptCBC(key, plaintext []byte) ([]byte, error) {
	block, err := sm4.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 生成随机IV
	iv := make([]byte, sm4.BlockSize)
	_, err = rand.Read(iv)
	if err != nil {
		return nil, err
	}

	// 对明文进行填充
	padding := sm4.BlockSize - len(plaintext)%sm4.BlockSize
	paddedPlaintext := append(plaintext, make([]byte, padding)...)
	for i := len(plaintext); i < len(paddedPlaintext); i++ {
		paddedPlaintext[i] = byte(padding)
	}

	ciphertext := make([]byte, len(paddedPlaintext))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, paddedPlaintext)

	// 将IV添加到密文前面
	ciphertext = append(iv, ciphertext...)

	return ciphertext, nil
}

// SM4DecryptCBC 使用SM4-CBC模式解密数据
func SM4DecryptCBC(key, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < sm4.BlockSize {
		return nil, errors.New("ciphertext too short")
	}

	block, err := sm4.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 分离IV和密文
	iv := ciphertext[:sm4.BlockSize]
	ciphertext = ciphertext[sm4.BlockSize:]

	plaintext := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(plaintext, ciphertext)

	// 移除填充
	padding := int(plaintext[len(plaintext)-1])
	if padding < 1 || padding > sm4.BlockSize {
		return nil, errors.New("invalid padding")
	}

	return plaintext[:len(plaintext)-padding], nil
}

// SM4EncryptCFB 使用SM4-CFB模式加密数据
func SM4EncryptCFB(key, plaintext []byte) ([]byte, error) {
	block, err := sm4.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 生成随机IV
	iv := make([]byte, sm4.BlockSize)
	_, err = rand.Read(iv)
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, len(plaintext))
	mode := cipher.NewCFBEncrypter(block, iv)
	mode.XORKeyStream(ciphertext, plaintext)

	// 将IV添加到密文前面
	ciphertext = append(iv, ciphertext...)

	return ciphertext, nil
}

// SM4DecryptCFB 使用SM4-CFB模式解密数据
func SM4DecryptCFB(key, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < sm4.BlockSize {
		return nil, errors.New("ciphertext too short")
	}

	block, err := sm4.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 分离IV和密文
	iv := ciphertext[:sm4.BlockSize]
	ciphertext = ciphertext[sm4.BlockSize:]

	plaintext := make([]byte, len(ciphertext))
	mode := cipher.NewCFBDecrypter(block, iv)
	mode.XORKeyStream(plaintext, ciphertext)

	return plaintext, nil
}

// SM4EncryptOFB 使用SM4-OFB模式加密数据
func SM4EncryptOFB(key, plaintext []byte) ([]byte, error) {
	block, err := sm4.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 生成随机IV
	iv := make([]byte, sm4.BlockSize)
	_, err = rand.Read(iv)
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, len(plaintext))
	mode := cipher.NewOFB(block, iv)
	mode.XORKeyStream(ciphertext, plaintext)

	// 将IV添加到密文前面
	ciphertext = append(iv, ciphertext...)

	return ciphertext, nil
}

// SM4DecryptOFB 使用SM4-OFB模式解密数据
func SM4DecryptOFB(key, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < sm4.BlockSize {
		return nil, errors.New("ciphertext too short")
	}

	block, err := sm4.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 分离IV和密文
	iv := ciphertext[:sm4.BlockSize]
	ciphertext = ciphertext[sm4.BlockSize:]

	plaintext := make([]byte, len(ciphertext))
	mode := cipher.NewOFB(block, iv)
	mode.XORKeyStream(plaintext, ciphertext)

	return plaintext, nil
}

// SM4EncryptCTR 使用SM4-CTR模式加密数据
func SM4EncryptCTR(key, plaintext []byte) ([]byte, error) {
	block, err := sm4.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 生成随机IV
	iv := make([]byte, sm4.BlockSize)
	_, err = rand.Read(iv)
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, len(plaintext))
	mode := cipher.NewCTR(block, iv)
	mode.XORKeyStream(ciphertext, plaintext)

	// 将IV添加到密文前面
	ciphertext = append(iv, ciphertext...)

	return ciphertext, nil
}

// SM4DecryptCTR 使用SM4-CTR模式解密数据
func SM4DecryptCTR(key, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < sm4.BlockSize {
		return nil, errors.New("ciphertext too short")
	}

	block, err := sm4.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 分离IV和密文
	iv := ciphertext[:sm4.BlockSize]
	ciphertext = ciphertext[sm4.BlockSize:]

	plaintext := make([]byte, len(ciphertext))
	mode := cipher.NewCTR(block, iv)
	mode.XORKeyStream(plaintext, ciphertext)

	return plaintext, nil
}

// SM4EncryptBase64 使用SM4-CBC模式加密数据并返回Base64编码
func SM4EncryptBase64(key []byte, plaintext string) (string, error) {
	ciphertext, err := SM4EncryptCBC(key, []byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// SM4DecryptBase64 解密Base64编码的SM4-CBC加密数据
func SM4DecryptBase64(key []byte, ciphertextBase64 string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", err
	}
	plaintext, err := SM4DecryptCBC(key, ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}