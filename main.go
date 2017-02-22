package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/fsnotify/fsnotify"
	yaml "gopkg.in/yaml.v2"
)

var (
	configFile  = flag.String("config.file", "fmufer.yml", "The fmufer configuration file.")
	showVersion = flag.Bool("version", false, "Print version information.")
)

type Config struct {
	Transfers []Transfer `yaml:"transfers"`
}

type Transfer struct {
	SrcDir   string `yaml:"src"`
	DestDir  string `yaml:"dst"`
	Pattern  string `yaml:"pattern"`
	Host     string `yaml:"host"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

func ResolvePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolvedPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", err
	}
	return resolvedPath, nil
}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Println("fmufer version 0.1")
		os.Exit(0)
	}

	log.Info("Starting fmufer")

	yamlFile, err := ioutil.ReadFile(*configFile)

	if err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}

	config := Config{}

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatalf("Error parsing config file: %s", err)
	}

	configs := make(map[string]Transfer)
	for _, t := range config.Transfers {
		absPath, err := ResolvePath(t.SrcDir)
		if err != nil {
			log.Fatal("could not creat config: ", err)
		}
		configs[absPath] = t
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				//log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					//log.Println("modified file:", event.Name)
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					log.Info("start transfer")
					srcDir, err := ResolvePath(filepath.Dir(event.Name))
					if err != nil {
						log.Error("could not get config: ", err)
						continue
					}
					transfer, ok := configs[srcDir]
					if !ok {
						log.Errorf("could not get config for '%s': %s", srcDir, err)
						continue
					}
					err = SftpTransfer(transfer, event.Name)
					if err != nil {
						log.Errorf("transfer of '%s' failed: %s", event.Name, err)
						continue
					}
				}
			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()

	for _, transfer := range config.Transfers {
		err = watcher.Add(transfer.SrcDir)
		if err != nil {
			log.Errorf("Failed to initialize directory watcher on '%s': %s", transfer.SrcDir, err)
		}
	}
	<-done

	log.Info("Stopping fmufer")

}
