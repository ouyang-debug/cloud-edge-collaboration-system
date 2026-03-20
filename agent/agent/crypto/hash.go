package crypto

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"io"
	"os"
)

// MD5 计算数据的MD5哈希值
func MD5(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

// MD5String 计算字符串的MD5哈希值
func MD5String(data string) string {
	return MD5([]byte(data))
}

// MD5File 计算文件的MD5哈希值
func MD5File(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// SHA1 计算数据的SHA1哈希值
func SHA1(data []byte) string {
	hash := sha1.Sum(data)
	return hex.EncodeToString(hash[:])
}

// SHA1String 计算字符串的SHA1哈希值
func SHA1String(data string) string {
	return SHA1([]byte(data))
}

// SHA1File 计算文件的SHA1哈希值
func SHA1File(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// SHA256 计算数据的SHA256哈希值
func SHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// SHA256String 计算字符串的SHA256哈希值
func SHA256String(data string) string {
	return SHA256([]byte(data))
}

// SHA256File 计算文件的SHA256哈希值
func SHA256File(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// SHA512 计算数据的SHA512哈希值
func SHA512(data []byte) string {
	hash := sha512.Sum512(data)
	return hex.EncodeToString(hash[:])
}

// SHA512String 计算字符串的SHA512哈希值
func SHA512String(data string) string {
	return SHA512([]byte(data))
}

// SHA512File 计算文件的SHA512哈希值
func SHA512File(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha512.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}