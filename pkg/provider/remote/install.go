package remote

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-getter"
	"github.com/nitrictech/cli/pkg/utils"
)

type ProviderInstall struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Version string `json:"version"`
}

func Install(provider *ProviderInstall) error {
	os.MkdirAll(filepath.Join(utils.NitricHomeDir(), "providers", provider.Name), 0777)
	dst := filepath.Join(utils.NitricHomeDir(), "providers", provider.Name, provider.Version)
	client := utils.NewGetter(&getter.Client{
		Ctx:  context.Background(),
		Dst:  dst,
		Src:  provider.URL,
		Mode: getter.ClientModeFile,
		Getters: map[string]getter.Getter{
			"https": &getter.HttpGetter{},
		},
	})

	// download file
	if err := client.Get(); err != nil {
		return err
	}

	if err := os.Chmod(dst, 0755); err != nil {
		return err
	}

	return os.WriteFile(dst+".origin", []byte(provider.URL), 0644)
}

func List() ([]ProviderInstall, error) {
	os.MkdirAll(filepath.Join(utils.NitricHomeDir(), "providers"), 0777)
	providers, err := os.ReadDir(filepath.Join(utils.NitricHomeDir(), "providers"))
	if err != nil {
		return nil, err
	}

	ps := []ProviderInstall{}

	for _, file := range providers {
		prov, err := os.ReadDir(filepath.Join(utils.NitricHomeDir(), "providers", file.Name()))
		if err != nil {
			return nil, err
		}

		for _, vfile := range prov {
			if strings.HasSuffix(vfile.Name(), ".origin") {
				continue
			}

			p := ProviderInstall{
				Name:    file.Name(),
				Version: vfile.Name(),
			}

			dat, err := os.ReadFile(filepath.Join(utils.NitricHomeDir(), "providers", file.Name(), vfile.Name()) + ".origin")
			if err == nil {
				p.URL = string(dat)
			}

			ps = append(ps, p)
		}
	}

	return ps, nil
}

func Remove(name string) error {
	return os.RemoveAll(filepath.Join(utils.NitricHomeDir(), "providers", name))
}
