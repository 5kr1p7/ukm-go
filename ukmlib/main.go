package ukm

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

const AGENT = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.0.0 Safari/537.36"

// Credentials
type Creds struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// UKM object
type UKM struct {
	Creds      *Creds
	IP         string
	Cookie     *http.Cookie
	HTTPClient *http.Client
}

// Kassa object
type Kassa struct {
	Name    string `json:"name,omitempty"`
	IP      string `json:"ip,omitempty"`
	Online  bool   `json:"online"`
	Version string `json:"version,omitempty"`
	Open    bool   `json:"open"`
	Cashier string `json:"cashier,omitempty"`
}

/*
   Check state by get presence of TD's attributes
*/
func getState(cell *html.Node) bool {
	if cell.FirstChild != nil && cell.FirstChild.Attr != nil {
		return true
	}

	return false
}

/*
   Get latest verion from UKM
*/
func getMaxVersion(table *html.Node) string {
	var ver_arr []string
	versions := htmlquery.Find(table, "//table/tbody/tr/td[6]")

	for _, ver := range versions {
		ver_arr = append(ver_arr, htmlquery.InnerText(ver))
	}

	sort.Strings(ver_arr)

	return ver_arr[0]
}

/*
   Login to UKM and get Cookie for futher requests
*/
func (ukm *UKM) login() (err error) {
	formData := url.Values{}
	formData.Add("LoginForm[username]", ukm.Creds.Username)
	formData.Add("LoginForm[password]", ukm.Creds.Password)
	body := strings.NewReader(formData.Encode())

	req, err := http.NewRequest("POST", "http://"+ukm.IP+"/ukm/index.php?r=site/login", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", AGENT)

	resp, err := ukm.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	ukm.Cookie = resp.Cookies()[1]

	return
}

/*
   Get list of cashboxes
*/
func (ukm *UKM) getKassaList() (string, error) {
	req, err := http.NewRequest("GET", "http://"+ukm.IP+"/ukm/index.php?r=pos/index&onlyGrid=1", nil)
	req.Header.Set("User-Agent", AGENT)
	req.AddCookie(ukm.Cookie)

	resp, err := ukm.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	doc, err := htmlquery.Parse(resp.Body)
	latest_ver := getMaxVersion(doc)

	rows := htmlquery.Find(doc, "//table/tbody/tr")

	// Fill array of shop's cashboxes
	cashboxes := []Kassa{}
	for _, row := range rows {
		kassa_n := Kassa{}
		cells := htmlquery.Find(row, "//td")

		// Check if cashbox have IP address and latest version
		if htmlquery.InnerText(cells[3]) != "" && htmlquery.InnerText(cells[5]) == latest_ver {
			kassa_n.Name = htmlquery.InnerText(cells[2])
			kassa_n.IP = htmlquery.InnerText(cells[3])
			kassa_n.Online = getState(cells[4])
			kassa_n.Open = getState(cells[6])
			kassa_n.Version = htmlquery.InnerText(cells[5])
			kassa_n.Cashier = htmlquery.InnerText(cells[8])

			cashboxes = append(cashboxes, kassa_n)
		}
	}

	outJSON, err := json.Marshal(cashboxes)
	if err != nil {
		return "", err
	}

	return string(outJSON), nil
}

/*
   Get Kassa list
*/
func (ukm *UKM) KassaList(w http.ResponseWriter, req *http.Request) {
	ip := req.URL.Query().Get("ip")
	log.Println("Get kassa list from", ip)

	if ip != "" && net.ParseIP(ip) != nil {
		ukm.IP = ip

		err := ukm.login()
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(err.Error()))
			log.Println("Error:", err.Error())
		} else {
			kassaList, err := ukm.getKassaList()
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(err.Error()))
			}

			fmt.Fprintf(w, kassaList)
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid IP address!"))
	}
}
