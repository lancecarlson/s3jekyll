package main

import (
	"launchpad.net/goamz/aws"
        "launchpad.net/goamz/s3"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"errors"
	"fmt"
	"os"
	"bufio"
	"mime"
	"strings"
	"flag"
)

var (
	ErrMissingAccess = errors.New("missing access key")
	ErrMissingSecret = errors.New("missing secret key")
	ErrMissingBucket = errors.New("missing bucket name")
	ErrMissingFrom = errors.New("missing from")
)

type Config struct {
	Access string `json:"access"`
	Secret string `json:"secret"`
	Bucket string `json:"bucket"`
	From string `json:"from"`
	To string `json:"to,omitempty"`
	Concurrency uint `json:"-"`
	Ignores []string `json:"ignores,omitempty"`
}

func (c *Config) Validate() (bool, error) {
	if c.Access == "" { return false, ErrMissingAccess }
	if c.Secret == "" { return false, ErrMissingSecret }
	if c.Bucket == "" { return false, ErrMissingBucket }
	if c.From == "" { return false, ErrMissingFrom }
	return true, nil
}

func (c *Config) Ignore(path string) (bool, error) {
	for _, ignore := range c.Ignores {
		match, err := filepath.Match(ignore, filepath.Base(path))
		if err != nil { return false, err }
		if match { return true, nil }
	}
	return false, nil
}

var BlankConfig = Config{"", "", "", "_site", "", 10, []string{}}

type ConfigFile struct {
	Path string
	File *os.File
	Config *Config
}

func (c *ConfigFile) OpenOrCreate(to string) (exists bool, err error) {
	err = c.setPath(to)
	if err != nil { return }

	exists, err = c.exists()
	if err != nil { return }
	if exists == true {
		err = c.open()
	} else {
		err = c.create()
	}
	c.File.Close()
	return 
}

func (c *ConfigFile) create() error {
	file, err := os.Create(c.Path)
	if err != nil { return err }
	c.File = file
	c.createBlank()
	return nil
}

func (c *ConfigFile) open() error {
	file, err := os.Open(c.Path)
	if err != nil { return err }
	c.File = file
	c.parse()
	return nil
}

func (c *ConfigFile) createBlank() error {
	b, err := json.MarshalIndent(BlankConfig, "", "    ")
	if err != nil { return err }
	return ioutil.WriteFile(c.Path, b, 0644)
}

func (c *ConfigFile) parse() (err error) {
	b, err := ioutil.ReadFile(c.Path)
	if err != nil { return }
	c.Config = new(Config)
	err = json.Unmarshal(b, c.Config)
	if err != nil { return }
	return
}

func (c *ConfigFile) exists() (bool, error) {
	_, err := os.Stat(c.Path)
	if err == nil { return true, nil }
	if os.IsNotExist(err) { return false, nil }
	return false, err
}

func (c *ConfigFile) setPath(to string) (err error) {
	pwd, err := os.Getwd()
	if err != nil { return }
	c.Path = pwd + "/." + to + ".s3.json"
	return
}

/*func PutFiles(bucket *s3.Bucket, config *Config, c chan string) {
	PutFile(bucket, config, p)
}*/

func PutFile(bucket *s3.Bucket, config *Config, path string) (err error) {
	stat, err := os.Stat(path)
	if err != nil { return }
	file, err := os.Open(path)
	if err != nil { return }
	defer file.Close()
	reader := bufio.NewReader(file)
	mimeType := mime.TypeByExtension(filepath.Ext(path))
	path = strings.Replace(path, config.From, "", 1)
	err = bucket.PutReader(config.To + path, reader, stat.Size(), mimeType, s3.PublicRead)
	if err != nil { return }
	return nil
}

func S3Walker(bucket *s3.Bucket, config *Config) filepath.WalkFunc {
	return func(path string, info os.FileInfo, inErr error) (err error) {
		if info.IsDir() {
			return nil
		}
		ignore, err := config.Ignore(path)
		if err != nil { return }
		if ignore == false {
			err = PutFile(bucket, config, path)
			if err != nil { return }
			fmt.Println(path)
		}
		return nil
	}
}

func main() {
	to := flag.String("to", "production", "name of the environment to push")
	concurrency := flag.Uint("n", 10, "number of concurrent uploads")
	flag.Parse()

	if *to == "" {
		flag.Usage()
		os.Exit(0)
	}

	cf := new(ConfigFile)
	exists, err := cf.OpenOrCreate(*to)
	if err != nil { panic(err) }
	if exists == false { panic("Config file didn't exist. Example created") }
	config := cf.Config
	config.Concurrency = *concurrency
	valid, err := config.Validate()
	if valid == false {
		fmt.Println(err)
		os.Exit(0)
	}

	auth := aws.Auth{AccessKey: config.Access, SecretKey: config.Secret}
	b := s3.New(auth, aws.USEast).Bucket(config.Bucket)

	filepath.Walk(config.From, S3Walker(b, config))
}