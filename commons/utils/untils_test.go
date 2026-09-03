package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNameValidation_AllowsValidName(t *testing.T) {
	warns, errs := ValidateName("ValidName", "")
	assert.Empty(t, warns)
	assert.Empty(t, errs)
}

func TestNameValidation_RejectsNameWithWhitespace(t *testing.T) {
	warns, errs := ValidateName("Invalid Name", "")
	assert.Empty(t, warns)
	assert.NotEmpty(t, errs)
	assert.Equal(t, "name cannot contain whitespace. Got Invalid Name", errs[0].Error())
}

func TestNameValidation_RejectsNonStringInput(t *testing.T) {
	warns, errs := ValidateName(123, "")
	assert.Empty(t, warns)
	assert.NotEmpty(t, errs)
	assert.Equal(t, "expected name to be string", errs[0].Error())
}

func TestSSHPublicKeyValidation_AllowsOpenSSHKey(t *testing.T) {
	warns, errs := ValidateSSHPublicKey("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDTUMz4TDTWt0W+vkt6v0VaOog28uoRjho1wkMx55055 terraform@example.com", "ssh_key")
	assert.Empty(t, warns)
	assert.Empty(t, errs)
}

func TestSSHPublicKeyValidation_AllowsRsaKeyWithTrailingNewline(t *testing.T) {
	warns, errs := ValidateSSHPublicKey("ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDtFxmsiEBQ/vyBdx2Elg6+OA7VFFVSNTzCq5pqXJ8Ao7wXEfJY/lfbfCUXUogt/+XxL6wY3Y3j14K6eY6EGieJtXv0SggZ6FH4FWPFD5Nd4+f0XQHBQnoYFHx4HHNd8E8ThIxdZ7s3q7qPZv3vYDPtgykgisRoWwOybPXbna57RCiCj4c5fmVTkRFa6nBAXk1l3Q7h+l/qnyvuIFC7ER8mpAWsNiYn+q4htA8I07umM31c8dA7iuyXbIMxxl6icfMTliEoB2y4RUc/Ts1ZrkzmsuUcnyqU0OYOaqVgLEKFxSqXBxDW21010Ve+ea3hCOdIo4WqopMBksE5PHXMC9X/ user@example.com\n", "ssh_key")
	assert.Empty(t, warns)
	assert.Empty(t, errs)
}

func TestSSHPublicKeyValidation_RejectsKeyName(t *testing.T) {
	warns, errs := ValidateSSHPublicKey("my-ssh-key", "ssh_key")
	assert.Empty(t, warns)
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0].Error(), "must be an OpenSSH public key")
}

func TestSSHPublicKeyValidation_RejectsPrivateKeyWithoutEchoingIt(t *testing.T) {
	warns, errs := ValidateSSHPublicKey("-----BEGIN OPENSSH PRIVATE KEY-----\nb3BlbnNzaC1rZXktdjEAAAAA\n-----END OPENSSH PRIVATE KEY-----", "ssh_key")
	assert.Empty(t, warns)
	assert.NotEmpty(t, errs)
	assert.Equal(t, "ssh_key must be a public key, not a private key: use the content of the matching .pub file", errs[0].Error())
}

func TestSSHPublicKeyValidation_RejectsMultipleKeys(t *testing.T) {
	warns, errs := ValidateSSHPublicKey("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIDTUMz4TDTWt0W+vkt6v0VaOog28uoRjho1wkMx55055 terraform@example.com\nssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDtFxmsiEBQ/vyBdx2Elg6+OA7VFFVSNTzCq5pqXJ8Ao7wXEfJY/lfbfCUXUogt/+XxL6wY3Y3j14K6eY6EGieJtXv0SggZ6FH4FWPFD5Nd4+f0XQHBQnoYFHx4HHNd8E8ThIxdZ7s3q7qPZv3vYDPtgykgisRoWwOybPXbna57RCiCj4c5fmVTkRFa6nBAXk1l3Q7h+l/qnyvuIFC7ER8mpAWsNiYn+q4htA8I07umM31c8dA7iuyXbIMxxl6icfMTliEoB2y4RUc/Ts1ZrkzmsuUcnyqU0OYOaqVgLEKFxSqXBxDW21010Ve+ea3hCOdIo4WqopMBksE5PHXMC9X/ user@example.com", "ssh_key")
	assert.Empty(t, warns)
	assert.NotEmpty(t, errs)
	assert.Equal(t, "ssh_key must contain exactly one public key", errs[0].Error())
}

func TestSSHPublicKeyValidation_RejectsNonStringInput(t *testing.T) {
	warns, errs := ValidateSSHPublicKey(123, "ssh_key")
	assert.Empty(t, warns)
	assert.NotEmpty(t, errs)
	assert.Equal(t, "expected ssh_key to be string", errs[0].Error())
}

func TestToQueryParams_ConvertsStructToQueryParams(t *testing.T) {
	type TestStruct struct {
		Name  string  `json:"name"`
		Age   int     `json:"age"`
		Score float64 `json:"score"`
	}
	data := TestStruct{Name: "John", Age: 30, Score: 95.5}
	expected := "?age=30&name=John&score=95.5"
	result := ToQueryParams(data)
	assert.Equal(t, expected, result)
}

func TestToQueryParams_HandlesEmptyStruct(t *testing.T) {
	type EmptyStruct struct{}
	data := EmptyStruct{}
	expected := "?"
	result := ToQueryParams(data)
	assert.Equal(t, expected, result)
}

func TestToQueryParams_HandlesNilPointer(t *testing.T) {
	type TestStruct struct {
		Name *string `json:"name"`
	}
	var name *string
	data := TestStruct{Name: name}
	expected := "?"
	result := ToQueryParams(data)
	assert.Equal(t, expected, result)
}

func TestToQueryParams_HandlesPointerFields(t *testing.T) {
	type TestStruct struct {
		Name *string `json:"name"`
	}
	name := "John"
	data := TestStruct{Name: &name}
	expected := "?name=John"
	result := ToQueryParams(data)
	assert.Equal(t, expected, result)
}

func TestCommaSeparatedAllowedKeys_ReturnsSortedKeys(t *testing.T) {
	keys := []string{"key3", "key1", "key2"}
	expected := "`key1`, `key2`, `key3`"
	result := GetCommaSeparatedAllowedKeys(keys)
	assert.Equal(t, expected, result)
}

func TestCommaSeparatedAllowedKeys_HandlesEmptyKeys(t *testing.T) {
	var keys []string
	expected := ""
	result := GetCommaSeparatedAllowedKeys(keys)
	assert.Equal(t, expected, result)
}
