package delugeclient

import (
	"golang.org/x/net/publicsuffix"
	"net/http/cookiejar"
	"net/http"
	"fmt"
	"log"
	"crypto/tls"
	"bytes"
	"encoding/json"
	"errors"
	"os"
)

type Deluge struct {
	ServiceUrl string
	Password   string
	Index      int
	HttpClient http.Client
}

type RpcError struct {
	Message string `json:"message"`
	Code    int `json:"code"`
}

type RpcResponse struct {
	Id     int        `json:"id"`
	Result bool        `json:"result"`
	Error  RpcError `json:"error"`
}

func (r RpcResponse)String() string {
	return fmt.Sprintf("id: '%d' result: '%s' error: {%s}", r.Id, r.Result, r.Error)
}
func (e RpcError)String() string {
	return fmt.Sprintf("code: '%d' message: '%s'", e.Code, e.Message)
}

func NewDeluge(serverUrl, password string) *Deluge {
	log.SetOutput(os.Stdout)
	options := cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	}
	cookieJar, err := cookiejar.New(&options)
	if err != nil {
		log.Fatal(err)
	}
	config := &tls.Config{InsecureSkipVerify: true}
	tr := &http.Transport{TLSClientConfig: config }
	return &Deluge{
		ServiceUrl:serverUrl + "/json",
		Password:password,
		Index:1,
		HttpClient:http.Client{Jar: cookieJar, Transport: tr},
	}
}

func (d *Deluge) Connect() error {

	var loginPayload = fmt.Sprintf(`{"id":%d, "method":"auth.login", "params":["%s"]}`, d.Index, d.Password)
	response, err := d.HttpClient.Post(d.ServiceUrl, "application/x-json", bytes.NewBufferString(loginPayload))
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if (response.StatusCode != 200) {
		return err
	}

	//fmt.Println("response Status:", response.Status)
	//fmt.Println("response Headers:", response.Header)
	//body, _ := ioutil.ReadAll(response.Body)
	//fmt.Println("response Body:", string(body))

	//rpcResponse := RpcResponse{}
	var rr RpcResponse
	json.NewDecoder(response.Body).Decode(&rr)
	if (!rr.Result) {
		log.Println(rr)
		return fmt.Errorf("Error code %d! %s.", rr.Error.Code, rr.Error.Message)
	}
	//serverUrl, _ := url.Parse("ip-94-23-205.eu")
	//cookie := d.HttpClient.Jar.Cookies(serverUrl)
	//fmt.Printf("cookies: %q\n", cookie)
	//fmt.Println(d.HttpClient.Jar)

	d.Index ++
	return nil
}

func (d *Deluge) AddMagnet(magnet string) error {
	//fmt.Println(d.HttpClient.Jar)
	//serverUrl, _ := url.Parse(d.ServiceUrl)
	//cookie := d.HttpClient.Jar.Cookies(serverUrl)
	//fmt.Printf("cookies: %q\n", cookie[0])
	var payload = fmt.Sprintf(`{"id":%d, "method":"web.add_torrents", "params":[[{"path":"%s", "options":""}]]}`, d.Index, magnet)
	response, err := d.HttpClient.Post(d.ServiceUrl, "application/x-json", bytes.NewBufferString(payload))

	//request, err := http.NewRequest("POST", d.ServiceUrl, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	//d.CookieJar.Cookies()
	//authCookie, _ :=request.Cookie("_session_id")
	//log.Println(string("cookie: " +authCookie.Value))
	//client := &http.Client{Transport:d.Transport, Jar:d.CookieJar}
	//response, err := client.Do(request)

	defer response.Body.Close()
	//fmt.Println("response Status:", response.Status)
	//fmt.Println("response Headers:", response.Header)
	//body, _ := ioutil.ReadAll(response.Body)
	//fmt.Println("response Body:", string(body))
	if (response.StatusCode != 200) {
		return fmt.Errorf("Server error response: %s.", response.Status)
	}

	var rr RpcResponse
	if err := json.NewDecoder(response.Body).Decode(&rr); err != nil {
		return errors.New("Unable to parse response body")
	}

	if (rr.Error.Code > 0) {
		log.Println(rr)
		return fmt.Errorf("Error code %d! %s.", rr.Error.Code, rr.Error.Message)
	}
	d.Index ++

	return nil
}
