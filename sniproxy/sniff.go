package sniproxy

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/powerpuffpenguin/streamf/pool"
)

func sniffSNI(pool *pool.Pool, r io.ReadCloser) (string, []byte, bool, error) {
	buf, closed, err := readClientHello(pool, r)
	if err != nil {
		return ``, buf, closed, err
	}
	sni, err := parseClientHello(buf)
	if err != nil {
		return ``, buf, false, err
	}
	return sni, buf, false, nil
}
func readClientHello(pool *pool.Pool, r io.ReadCloser) ([]byte, bool, error) {
	buf := pool.Get()
	// 讀取 TLS Record Header (5 bytes)
	header := buf[:5]
	if n, err := io.ReadFull(r, header); err != nil {
		r.Close()
		return header[:n], true, errors.New("failed to read TLS record header: " + err.Error())
	}

	// 檢查是否為 TLS Handshake 記錄 (type 0x16)
	if header[0] != 0x16 {
		return header, false, errors.New("not a TLS handshake record")
	}

	// 提取記錄長度
	length := int(binary.BigEndian.Uint16(header[3:5]))
	if length < 38 { // 最小 Client Hello 長度（不含記錄頭部）
		return header, false, errors.New("TLS record too short")
	}

	// 分配緩衝區並將頭部複製進去
	size := 5 + length
	if len(buf) >= size {
		buf = buf[:size]
	} else {
		buf = make([]byte, size)
		copy(buf, header)
	}

	// 讀取剩餘的記錄內容
	if n, err := io.ReadFull(r, buf[5:]); err != nil {
		r.Close()
		return buf[:5+n], true, errors.New("failed to read TLS record body: " + err.Error())
	}

	// 檢查是否為 Client Hello (Handshake Type 0x01)
	if len(buf) < 6 || buf[5] != 0x01 {
		return buf, false, errors.New("not a Client Hello message")
	}

	return buf, false, nil
}
func parseClientHello(buf []byte) (string, error) {
	// 檢查數據長度是否足夠
	if len(buf) < 5 {
		return "", errors.New("buffer too short")
	}

	// 檢查是否為 TLS Handshake 訊息
	if buf[0] != 0x16 { // Handshake record type
		return "", errors.New("not a TLS handshake")
	}

	// 檢查是否為 Client Hello (type 1)
	if len(buf) < 43 || buf[5] != 0x01 {
		return "", errors.New("not a Client Hello message")
	}

	// 跳過固定長度的頭部
	// Record Header (5 bytes) + Handshake Header (4 bytes) + Version (2 bytes) + Random (32 bytes) = 43 bytes
	pos := 43

	// 跳過 Session ID
	sessionIDLen := int(buf[pos])
	pos += 1 + sessionIDLen
	if pos >= len(buf) {
		return "", errors.New("invalid session ID length")
	}

	// 跳過 Cipher Suites
	cipherSuitesLen := int(binary.BigEndian.Uint16(buf[pos:]))
	pos += 2 + cipherSuitesLen
	if pos >= len(buf) {
		return "", errors.New("invalid cipher suites length")
	}

	// 跳過 Compression Methods
	compMethodsLen := int(buf[pos])
	pos += 1 + compMethodsLen
	if pos >= len(buf) {
		return "", errors.New("invalid compression methods length")
	}

	// 檢查是否有 Extensions
	if pos+2 > len(buf) {
		return "", errors.New("no extensions found")
	}

	// 讀取 Extensions 長度
	extensionsLen := int(binary.BigEndian.Uint16(buf[pos:]))
	pos += 2
	if pos+extensionsLen > len(buf) {
		return "", errors.New("invalid extensions length")
	}

	// 解析 Extensions
	end := pos + extensionsLen
	for pos < end {
		if pos+4 > len(buf) {
			return "", errors.New("invalid extension format")
		}

		// 讀取 Extension Type 和 Length
		extType := binary.BigEndian.Uint16(buf[pos:])
		extLen := binary.BigEndian.Uint16(buf[pos+2:])
		pos += 4

		if pos+int(extLen) > len(buf) {
			return "", errors.New("invalid extension length")
		}

		// 檢查是否為 SNI Extension (type 0)
		if extType == 0 {
			// 跳過 Server Name List 長度 (2 bytes)
			if pos+2 > len(buf) {
				return "", errors.New("invalid SNI format")
			}
			nameListLen := binary.BigEndian.Uint16(buf[pos:])
			pos += 2

			if pos+int(nameListLen) > len(buf) {
				return "", errors.New("invalid SNI list length")
			}

			// 檢查 Name Type (通常為 0，表示 host_name)
			if pos+1 > len(buf) || buf[pos] != 0 {
				return "", errors.New("invalid SNI name type")
			}
			pos++

			// 讀取 Server Name 長度
			if pos+2 > len(buf) {
				return "", errors.New("invalid SNI name length")
			}
			nameLen := binary.BigEndian.Uint16(buf[pos:])
			pos += 2

			if pos+int(nameLen) > len(buf) {
				return "", errors.New("invalid SNI name data")
			}

			// 提取 SNI
			sni := string(buf[pos : pos+int(nameLen)])
			return sni, nil
		}

		// 跳到下一個 Extension
		pos += int(extLen)
	}

	return "", errors.New("SNI not found")
}
