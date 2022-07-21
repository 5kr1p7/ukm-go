package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/5kr1p7/ukm"
	"github.com/go-yaml/yaml"
)

/*
   Load credentials from YAML config
*/
func loadConfig(confFile string) *ukm.Creds {
	confContent, err := ioutil.ReadFile(confFile)
	if err != nil {
		log.Fatalln(err.Error())
	}

	conf := &ukm.Creds{}
	if err := yaml.Unmarshal(confContent, conf); err != nil {
		log.Fatalln(err.Error())
	}

	return conf
}

const PORT = ":4433"

// -------------------------------------

func main() {
	conf := loadConfig("ukm.yaml")

	serv := ukm.UKM{
		Creds: conf,
		HTTPClient: &http.Client{
			Timeout: time.Second * 3,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}

	http.HandleFunc("/ukm", serv.KassaList)

	log.Printf("Listening on %s...", PORT)
	log.Fatal(http.ListenAndServe(PORT, nil))
}
