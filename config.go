package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	data         configData
	configFolder string
}

type configData struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	UserName     string `json:"user_name"`
}

func (c *Config) SetConfigFolder(name string) error {
	confdir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	abspath := filepath.Join(confdir, name)
	err = os.MkdirAll(abspath, os.ModePerm)
	if err != nil {
		return err
	}
	c.configFolder = abspath
	return nil
}

func (c *Config) Load(filename string) error {
	if c.configFolder == "" {
		return errors.New("config folder not set")
	}
	file, err := os.Open(filepath.Join(c.configFolder, filename))
	if err != nil {
		return err
	}
	defer file.Close()
	dec := json.NewDecoder(file)
	err = dec.Decode(&c.data)
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) GetFromUserInput() error {
	rdr := bufio.NewReader(os.Stdin)
	fmt.Print("Please input Client ID: ")
	clientID, err := rdr.ReadString('\n')
	if err != nil {
		fmt.Println()
		return err
	}
	c.data.ClientID = strings.TrimSpace(clientID)
	fmt.Print("Please input Client Secret: ")
	clientSecret, err := rdr.ReadString('\n')
	if err != nil {
		fmt.Println()
		return err
	}
	c.data.ClientSecret = strings.TrimSpace(clientSecret)
	fmt.Print("Please input User Name: ")
	userName, err := rdr.ReadString('\n')
	if err != nil {
		fmt.Println()
		return err
	}
	c.data.UserName = strings.TrimSpace(userName)
	return nil
}

func (c *Config) Save(filename string) error {
	if c.configFolder == "" {
		return errors.New("config folder not set")
	}
	file, err := os.Create(filepath.Join(c.configFolder, filename))
	if err != nil {
		return err
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	enc.SetIndent("", "    ")
	enc.Encode(c.data)
	return nil
}
