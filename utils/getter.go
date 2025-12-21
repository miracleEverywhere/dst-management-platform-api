package utils

import (
	"encoding/base64"
	"encoding/hex"
)

func GetSteamApiKey() string {
	obfuscated := []byte{
		0xD5, 0xED, 0xDA, 0x66, 0x64, 0xFF, 0x23, 0xA6,
		0xB3, 0xD8, 0x50, 0x2C, 0x63, 0xB1, 0xBF, 0x6D,
	}
	var data []byte
	for _, b := range obfuscated {
		data = append(data, b^0x55)
	}
	return hex.EncodeToString(data)
}

func GetDstToken() string {
	decoded := "VjFSQ2ExVXlWbkpsUm1oaFVqRmFWVlJXV21GaVZsWnlZVWRHV0dKV1JqUldNakI0V1ZaS1YyTkhlRlpOVjFKeVZrUkJOVkpzY0VkYVJtaFVVakpSZVZac1dsTlNNazE0VW14a1VtSlZXbWhVVlZKelUyeHJlRlZyT1ZaaVJscEpWMnRTUzFac1NYbFVXSEJhWld0YWRsa3haRWRYVms1VlZHeGtWMDFZUWtoV01qRjNZbTFXV0Zac1dtcFNSVXB2V2xkd1FrOVJQVDA9"
	for i := 0; i < 5; i++ {
		data, _ := base64.StdEncoding.DecodeString(decoded)
		decoded = string(data)
	}

	return decoded
}
