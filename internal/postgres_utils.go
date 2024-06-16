package foodme

import "bytes"

func calculatePacketSize(sizebuff []byte) int {
	return int(sizebuff[0])<<24 | int(sizebuff[1])<<16 | int(sizebuff[2])<<8 | int(sizebuff[3])
}

func createPacketSize(size int) []byte {
	return []byte{byte(size >> 24), byte(size >> 16), byte(size >> 8), byte(size)}
}

func checkAuthenticationSuccess(r []byte) bool {
	if len(r) != 4 {
		return false
	}
	if r[0] != 0 || r[1] != 0 || r[2] != 0 || r[3] != 0 {
		return false
	}
	return true
}

func getErrorMessage(data []byte) string {
	parts := bytes.Split(data, []byte{0})
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		if p[0] == 'M' {
			return string(p[1:])
		}
	}
	return "unknown error"
}
