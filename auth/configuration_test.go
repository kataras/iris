package auth

import (
	"fmt"
	"testing"

	"github.com/kataras/jwt"
)

func TestConfiguration_BindRandom(t *testing.T) {
	var c Configuration
	if err := c.BindRandom(); err != nil {
		t.Fatal(err)
	}

	if _, ok := c.Keys.Get(KIDAccess); !ok {
		t.Fatalf("expected key with kid: %s", KIDAccess)
	}

	refreshKeyConfig, ok := c.Keys.Get(KIDRefresh)
	if !ok {
		t.Fatalf("expected key with kid: %s", KIDRefresh)
	}

	encrypt, decrypt, err := jwt.GCM([]byte(refreshKeyConfig.EncryptionKey), nil)
	if err != nil {
		t.Fatal(err)
	}

	var data = []byte("some-data")
	encrypted, err := encrypt(data)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	decrypted, err := decrypt(encrypted)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}

	if expected, got := string(decrypted), string(data); expected != got {
		t.Fatalf("expected decrypted data to be equal to original data, expected: %s, got: %s", expected, got)
	}
	// Print the configuration as YAML.
	yamlConfig, err := c.ToYAML()
	if err != nil {
		panic(err)
	}
	fmt.Println(string(yamlConfig))
}
