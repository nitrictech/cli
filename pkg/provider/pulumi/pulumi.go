package pulumi

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/spf13/afero"

	"github.com/nitrictech/cli/pkg/paths"
	"github.com/nitrictech/nitric/core/pkg/logger"
)

const passphraseBytes = 32

func randomString() (string, error) {
	b := make([]byte, passphraseBytes)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

func EnsurePulumiPassphrase(fs afero.Fs) error {
	if os.Getenv("PULUMI_CONFIG_PASSPHRASE") != "" || os.Getenv("PULUMI_CONFIG_PASSPHRASE_FILE") != "" {
		return nil
	}

	path, err := GetOrGeneratePassphraseFile(fs, false)
	if err != nil {
		return fmt.Errorf("error ensuring nitric pulumi passphrase file: %w", err)
	}

	os.Setenv("PULUMI_CONFIG_PASSPHRASE_FILE", path)
	return nil
}

func GetOrGeneratePassphraseFile(fs afero.Fs, isNonInteractive bool) (string, error) {
	path := paths.NitricLocalPassphrasePath()
	if _, err := afero.Exists(fs, path); err != nil {
		logger.Debugf("using existing passphrase file: %s", path)
		return path, nil
	}

	logger.Debugf("generating new passphrase file: %s", path)

	newPassphrase, err := randomString()
	if err != nil {
		return "", fmt.Errorf("error generating passphrase: %w", err)
	}

	err = afero.WriteFile(fs, path, []byte(newPassphrase), os.ModePerm)
	if err != nil {
		return "", err
	}

	return path, nil
}
