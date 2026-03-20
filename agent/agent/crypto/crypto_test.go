package crypto

import (
	"fmt"
	"testing"
)

func TestGenerateRSAKeyPair(t *testing.T) {
	privateKey, publicKey, err := GenerateRSAKeyPair(2048)
	if err != nil {
		t.Errorf("GenerateRSAKeyPair failed: %v", err)
	}

	if privateKey == nil || publicKey == nil {
		t.Error("GenerateRSAKeyPair returned nil key")
	}

	// 测试PEM转换
	privateKeyPEM := PrivateKeyToPEM(privateKey)
	publicKeyPEM := PublicKeyToPEM(publicKey)

	if privateKeyPEM == nil || publicKeyPEM == nil {
		t.Error("PEM conversion failed")
	}

	// 测试PEM解析
	parsedPrivateKey, err := PEMToPrivateKey(privateKeyPEM)
	if err != nil {
		t.Errorf("PEMToPrivateKey failed: %v", err)
	}

	parsedPublicKey, err := PEMToPublicKey(publicKeyPEM)
	if err != nil {
		t.Errorf("PEMToPublicKey failed: %v", err)
	}

	if parsedPrivateKey == nil || parsedPublicKey == nil {
		t.Error("PEM parsing failed")
	}
}

func TestRSAEncryptDecrypt(t *testing.T) {
	privateKey, publicKey, err := GenerateRSAKeyPair(2048)
	if err != nil {
		t.Errorf("GenerateRSAKeyPair failed: %v", err)
		return
	}

	originalText := []byte("Hello, World!")

	// 加密
	encrypted, err := RSAEncrypt(publicKey, originalText)
	if err != nil {
		t.Errorf("RSAEncrypt failed: %v", err)
		return
	}

	// 解密
	decrypted, err := RSADecrypt(privateKey, encrypted)
	if err != nil {
		t.Errorf("RSADecrypt failed: %v", err)
		return
	}

	if string(decrypted) != string(originalText) {
		t.Errorf("Decrypted text mismatch, got: %s, want: %s", decrypted, originalText)
	}
}

func TestRSASignVerify(t *testing.T) {
	privateKey, publicKey, err := GenerateRSAKeyPair(2048)
	if err != nil {
		t.Errorf("GenerateRSAKeyPair failed: %v", err)
		return
	}

	data := []byte("Hello, World!")

	// 签名
	signature, err := RSASign(privateKey, data)
	if err != nil {
		t.Errorf("RSASign failed: %v", err)
		return
	}

	// 验签
	err = RSAVerify(publicKey, data, signature)
	if err != nil {
		t.Errorf("RSAVerify failed: %v", err)
	}
}

func TestGenerateAESKey(t *testing.T) {
	key, err := GenerateAESKey(256)
	if err != nil {
		t.Errorf("GenerateAESKey failed: %v", err)
	}

	if len(key) != 32 {
		t.Errorf("AES key length mismatch, got: %d, want: 32", len(key))
	}

	// 测试Base64转换
	keyBase64 := AESKeyToBase64(key)
	parsedKey, err := Base64ToAESKey(keyBase64)
	if err != nil {
		t.Errorf("Base64ToAESKey failed: %v", err)
	}

	if string(parsedKey) != string(key) {
		t.Error("Base64 conversion mismatch")
	}
}

func TestAESEncryptDecrypt(t *testing.T) {
	key, err := GenerateAESKey(256)
	if err != nil {
		t.Errorf("GenerateAESKey failed: %v", err)
		return
	}

	originalText := []byte("Hello, World!")

	// 加密
	encrypted, err := AESEncrypt(key, originalText)
	if err != nil {
		t.Errorf("AESEncrypt failed: %v", err)
		return
	}

	// 解密
	decrypted, err := AESDecrypt(key, encrypted)
	if err != nil {
		t.Errorf("AESDecrypt failed: %v", err)
		return
	}

	if string(decrypted) != string(originalText) {
		t.Errorf("Decrypted text mismatch, got: %s, want: %s", decrypted, originalText)
	}
}

func TestHashFunctions(t *testing.T) {
	data := "Hello, World!"

	// 测试MD5
	md5Result := MD5String(data)
	if md5Result == "" {
		t.Error("MD5String failed")
	}

	// 测试SHA1
	sha1Result := SHA1String(data)
	if sha1Result == "" {
		t.Error("SHA1String failed")
	}

	// 测试SHA256
	sha256Result := SHA256String(data)
	if sha256Result == "" {
		t.Error("SHA256String failed")
	}

	// 测试SHA512
	sha512Result := SHA512String(data)
	if sha512Result == "" {
		t.Error("SHA512String failed")
	}
}

// ------------ SM4算法测试 ------------

func TestGenerateSM4Key(t *testing.T) {
	key, err := GenerateSM4Key()
	if err != nil {
		t.Errorf("GenerateSM4Key failed: %v", err)
	}

	if len(key) != 16 {
		t.Errorf("SM4 key length mismatch, got: %d, want: 16", len(key))
	}

	// 测试Base64转换
	keyBase64 := SM4KeyToBase64(key)
	parsedKey, err := Base64ToSM4Key(keyBase64)
	if err != nil {
		t.Errorf("Base64ToSM4Key failed: %v", err)
	}

	if string(parsedKey) != string(key) {
		t.Error("Base64 conversion mismatch")
	}
}

func TestSM4EncryptDecryptECB(t *testing.T) {
	key, err := GenerateSM4Key()
	if err != nil {
		t.Errorf("GenerateSM4Key failed: %v", err)
		return
	}

	originalText := []byte("Hello, World!")

	// 加密
	encrypted, err := SM4EncryptECB(key, originalText)
	if err != nil {
		t.Errorf("SM4EncryptECB failed: %v", err)
		return
	}

	// 解密
	decrypted, err := SM4DecryptECB(key, encrypted)
	if err != nil {
		t.Errorf("SM4DecryptECB failed: %v", err)
		return
	}

	if string(decrypted) != string(originalText) {
		t.Errorf("Decrypted text mismatch, got: %s, want: %s", decrypted, originalText)
	}
}

func TestSM4EncryptDecryptCBC(t *testing.T) {
	key, err := GenerateSM4Key()
	if err != nil {
		t.Errorf("GenerateSM4Key failed: %v", err)
		return
	}

	originalText := []byte("Hello, World!")

	// 加密
	encrypted, err := SM4EncryptCBC(key, originalText)
	if err != nil {
		t.Errorf("SM4EncryptCBC failed: %v", err)
		return
	}

	// 解密
	decrypted, err := SM4DecryptCBC(key, encrypted)
	if err != nil {
		t.Errorf("SM4DecryptCBC failed: %v", err)
		return
	}

	if string(decrypted) != string(originalText) {
		t.Errorf("Decrypted text mismatch, got: %s, want: %s", decrypted, originalText)
	}
}

func TestSM4EncryptDecryptCFB(t *testing.T) {
	key, err := GenerateSM4Key()
	if err != nil {
		t.Errorf("GenerateSM4Key failed: %v", err)
		return
	}

	originalText := []byte("Hello, World!")

	// 加密
	encrypted, err := SM4EncryptCFB(key, originalText)
	if err != nil {
		t.Errorf("SM4EncryptCFB failed: %v", err)
		return
	}

	// 解密
	decrypted, err := SM4DecryptCFB(key, encrypted)
	if err != nil {
		t.Errorf("SM4DecryptCFB failed: %v", err)
		return
	}

	if string(decrypted) != string(originalText) {
		t.Errorf("Decrypted text mismatch, got: %s, want: %s", decrypted, originalText)
	}
}

func TestSM4EncryptDecryptOFB(t *testing.T) {
	key, err := GenerateSM4Key()
	if err != nil {
		t.Errorf("GenerateSM4Key failed: %v", err)
		return
	}

	originalText := []byte("Hello, World!")

	// 加密
	encrypted, err := SM4EncryptOFB(key, originalText)
	if err != nil {
		t.Errorf("SM4EncryptOFB failed: %v", err)
		return
	}

	// 解密
	decrypted, err := SM4DecryptOFB(key, encrypted)
	if err != nil {
		t.Errorf("SM4DecryptOFB failed: %v", err)
		return
	}

	if string(decrypted) != string(originalText) {
		t.Errorf("Decrypted text mismatch, got: %s, want: %s", decrypted, originalText)
	}
}

func TestSM4EncryptDecryptCTR(t *testing.T) {
	key, err := GenerateSM4Key()
	if err != nil {
		t.Errorf("GenerateSM4Key failed: %v", err)
		return
	}

	originalText := []byte("Hello, World!")

	// 加密
	encrypted, err := SM4EncryptCTR(key, originalText)
	if err != nil {
		t.Errorf("SM4EncryptCTR failed: %v", err)
		return
	}

	// 解密
	decrypted, err := SM4DecryptCTR(key, encrypted)
	if err != nil {
		t.Errorf("SM4DecryptCTR failed: %v", err)
		return
	}

	if string(decrypted) != string(originalText) {
		t.Errorf("Decrypted text mismatch, got: %s, want: %s", decrypted, originalText)
	}
}

func TestSM4EncryptDecryptBase64(t *testing.T) {
	key, err := GenerateSM4Key()
	if err != nil {
		t.Errorf("GenerateSM4Key failed: %v", err)
		return
	}

	keyBase64 := SM4KeyToBase64(key)
	// keyBase64
	fmt.Printf("base64: encrypted: %s\n", keyBase64)

	parsedKey, err := Base64ToSM4Key(keyBase64)
	if err != nil {
		t.Errorf("Base64ToSM4Key failed: %v", err)
	}

	originalText := "Hello, World!"

	// 加密
	encrypted, err := SM4EncryptBase64(key, originalText)
	if err != nil {
		t.Errorf("SM4EncryptBase64 failed: %v", err)
		return
	}

	// 解密
	decrypted, err := SM4DecryptBase64(parsedKey, encrypted)
	if err != nil {
		t.Errorf("SM4DecryptBase64 failed: %v", err)
		return
	}

	if decrypted != originalText {
		t.Errorf("Decrypted text mismatch, got: %s, want: %s", decrypted, originalText)
	}
}
