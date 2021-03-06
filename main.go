package main

import (
	"bufio"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	yaml "gopkg.in/yaml.v2"
	"io"
	"os"
)

// SecretData extracts out the data portion of a kubernetes secret
type SecretData struct {
	Data map[string]string `json:"data" yaml:"data"`
}

// Secret allows us to read and return the full kubernetes secret
type Secret map[string]interface{}

// Unmarshallable allows me to unmarsal different strings with the same interface
type Unmarshallable func([]byte, interface{}) error

func main() {
	info, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}

	if (info.Mode()&os.ModeCharDevice) != 0 || info.Size() < 0 {
		fmt.Println("The command is intended to work with pipes.")
		fmt.Println("Usage: kubectl get secret <secret-name> -o <yaml|json> |", os.Args[0])
		return
	}

	output := getKubectlSecretOutput()
	isJSON := isJSON(output)
	unmarshal := getUnmarshalByOutputType(isJSON)

	sd, err := getDecodedSecretData(unmarshal, output)
	if err != nil {
		fmt.Print(err)
		return
	}

	s, err := getFullSecretWithDecodedData(unmarshal, output, sd)
	if err != nil {
		fmt.Print(err)
		return
	}

	secret := getStringSecret(s, isJSON)

	// Print exposed secret
	fmt.Print(secret)
}

func getUnmarshalByOutputType(isJSON bool) Unmarshallable {

	var unmarshal Unmarshallable
	if isJSON {
		unmarshal = json.Unmarshal
	} else {
		unmarshal = yaml.Unmarshal
	}

	return unmarshal
}

func getStringSecret(s *Secret, isJSON bool) string {
	var secret []byte
	if isJSON {
		secret, _ = json.MarshalIndent(s, "", "    ")
	} else {
		secret, _ = yaml.Marshal(s)
	}

	return string(secret)
}

func getFullSecretWithDecodedData(unmarshal Unmarshallable, output []byte, sd *SecretData) (*Secret, error) {
	var s Secret
	var err error

	err = unmarshal(output, &s)
	if err != nil {
		return nil, err
	}

	for key, _ := range s {
		if key == "data" {
			s[key] = sd.Data
		}
	}

	return &s, nil
}

func getDecodedSecretData(unmarshal Unmarshallable, output []byte) (*SecretData, error) {
	var s SecretData
	var err error

	err = unmarshal(output, &s)
	if err != nil {
		return nil, err
	}

	err = parseData(&s)
	if err != nil {
		return nil, err
	}

	return &s, nil
}

func getKubectlSecretOutput() []byte {
	reader := bufio.NewReader(os.Stdin)
	var output []byte

	for {
		input, err := reader.ReadByte()
		if err != nil && err == io.EOF {
			break
		}

		output = append(output, input)
	}

	return output
}

func isJSON(s []byte) bool {
	var js interface{}
	return json.Unmarshal(s, &js) == nil
}

func parseData(s *SecretData) error {
	var err error
	for key, value := range s.Data {
		s.Data[key], err = decodeString(value)
		if err != nil {
			return err
		}
	}

	return nil
}

func decodeString(encoded string) (string, error) {
	decoded, err := b64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	return string(decoded), nil
}
