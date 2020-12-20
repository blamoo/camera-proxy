package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

var config Config

type Camera struct {
	Name  string
	URL   string
	Label string
}

func (c *Camera) ViewURL() string {
	return fmt.Sprintf("/camera/%s", c.Name)
}

type Config struct {
	Debug               bool
	ServerHost          string
	ServerPort          int
	SessionCookieName   string
	SessionCookieMaxAge int
	TLSCertFile         string
	TLSKeyFile          string
	SessionCookieKey    []byte
	Users               map[string]string
	Cameras             []Camera
}

func InitializeConfig() error {
	var err error
	configFile, err := os.Open(configPath)

	if err != nil {
		return err
	}

	configBytes, err := ioutil.ReadAll(configFile)
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

	return nil
}
