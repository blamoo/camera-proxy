package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
)

var config Config

type Camera struct {
	Name  string
	URL   string
	Label string
	Files string
}

func (c Camera) ViewURL() string {
	return fmt.Sprintf("/camera/%s", c.Name)
}

func (c Camera) FileURL(f ...string) string {
	j := filepath.Join(f...)
	j = strings.ReplaceAll(j, "\\", "/")
	return fmt.Sprintf("/camera/files/%s/%s", c.Name, j)
}

type Config struct {
	Debug               bool
	ServerHost          string
	ServerPort          int
	LocalHost           string
	LocalPort           int
	SessionCookieName   string
	SessionCookieMaxAge int
	TLSCertFile         string
	TLSKeyFile          string
	SessionCookieKey    []byte
	Users               map[string]string
	Cameras             []Camera
	AuthWhitelist       []string
}

func (c Config) FindCamera(name string) (Camera, error) {
	for _, camera := range c.Cameras {
		if name == camera.Name {
			return camera, nil
		}
	}

	return Camera{}, fmt.Errorf("Câmera não encontrada")
}

func InitializeConfig() error {
	var err error
	configFile, err := os.Open(configPath)

	if err != nil {
		return err
	}

	configBytes, err := io.ReadAll(configFile)
	if err != nil {
		return err
	}

	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		return err
	}

	_, err = os.Stat(config.TLSCertFile)
	if err != nil {
		return fmt.Errorf("Erro ao carregar TLSCertFile. Caminho informado: '%s'. Erro: %s", config.TLSCertFile, err)
	}

	_, err = os.Stat(config.TLSKeyFile)
	if err != nil {
		return fmt.Errorf("Erro ao carregar TLSKeyFile. Caminho informado: '%s'. Erro: %s", config.TLSKeyFile, err)
	}

	if config.SessionCookieMaxAge <= 0 {
		return fmt.Errorf("SessionMaxAge deve ser maior que 0")
	}

	if len(config.SessionCookieKey) < 32 {
		fmt.Println("SessionKey deve ser uma sequência de bytes secerta com no mínimo 32 bytes codificada em base64")
		fmt.Println("Como essa aqui:")
		b := make([]byte, 32)
		rand.Read(b)
		sEnc := base64.StdEncoding.EncodeToString(b)
		fmt.Println(sEnc)
		return fmt.Errorf("Atualize seu arquivo de configuração com um valor válido e execute a aplicação novamente")
	}

	for _, cidr := range config.AuthWhitelist {
		_, _, err := net.ParseCIDR(cidr)

		if err != nil {
			return fmt.Errorf("Erro ao carregar whitelist de IPs. Formato correto: 192.168.0.0/24 Formato usado: '%s'. Erro: %s", cidr, err)
		}
	}

	return nil
}
