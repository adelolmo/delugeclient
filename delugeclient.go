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
	Id     int `json:"id"`
	Result bool `json:"result"`
	Error  RpcError `json:"error"`
}

func (r RpcResponse)String() string {
	return fmt.Sprintf("id: '%d' result: '%s' error: {%s}", r.Id, r.Result, r.Error)
}
func (e RpcError)String() string {
	return fmt.Sprintf("code: '%d' message: '%s'", e.Code, e.Message)
}

type Torrent struct {
	Id         string
	Name       string
	ShareRatio float64
}

type TorrentEntry struct {
	Message string `json:"message"`
	Ratio   float64 `json:"ratio"`
	Name    string `json:"name"`
}

type TorrentSet struct {
	Map map[string]TorrentEntry `json:"torrents"`
}

type AllResponse struct {
	Index    int `json:"id"`
	Torrents TorrentSet `json:"result"`
	Error    RpcError `json:"error"`
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
	var payload = fmt.Sprintf(`{"id":%d, "method":"auth.login", "params":["%s"]}`, d.Index, d.Password)
	var rr RpcResponse
	err := SendRequest(d.HttpClient, d.ServiceUrl, payload, &rr)

	if err != nil {
		return err
	}
	if (!rr.Result) {
		return fmt.Errorf("Error code %d! %s.", rr.Error.Code, rr.Error.Message)
	}

	d.Index ++
	return nil
}

func (d *Deluge) AddMagnet(magnet string) error {
	var payload = fmt.Sprintf(`{"id":%d, "method":"web.add_torrents", "params":[[{"path":"%s", "options":""}]]}`, d.Index, magnet)
	var rr RpcResponse
	err := SendRequest(d.HttpClient, d.ServiceUrl, payload, &rr)

	if err != nil {
		return err
	}
	if (rr.Error.Code > 0) {
		log.Println(rr)
		return fmt.Errorf("Error code %d! %s.", rr.Error.Code, rr.Error.Message)
	}
	d.Index ++
	return nil
}

func (d *Deluge) GetAll() ([]Torrent, error) {
	var payload = fmt.Sprintf(`{"id":%d, "method":"web.update_ui", "params":[["name", "ratio", "message"],{}]}`, d.Index)
	var rr AllResponse
	err := SendRequest(d.HttpClient, d.ServiceUrl, payload, &rr)
	if (err != nil) {
		panic(err)
	}
	if (rr.Error.Code > 0) {
		log.Println(rr)
		return nil, fmt.Errorf("Error code %d! %s.", rr.Error.Code, rr.Error.Message)
	}

	var torrents = make([]Torrent, len(rr.Torrents.Map))

	var index = 0
	for k, v := range rr.Torrents.Map {
		torrents[index] = Torrent{Id:k, Name:v.Name, ShareRatio:v.Ratio}
		index++
	}
	d.Index ++
	return torrents, nil
}

func (d *Deluge) Remove(torrentId string) error {
	var payload = fmt.Sprintf(`{"id":%d, "method":"core.remove_torrent", "params":["%s",true]}`, d.Index, torrentId)
	var rr RpcResponse
	err := SendRequest(d.HttpClient, d.ServiceUrl, payload, &rr)
	if err != nil {
		return err
	}

	if (rr.Error.Code > 0) {
		log.Println(rr)
		return fmt.Errorf("Error code %d! %s.", rr.Error.Code, rr.Error.Message)
	}
	d.Index ++
	return nil
}

func SendRequest(httpClient http.Client, url, payload string, decoder interface{}) error {
	response, err := httpClient.Post(url, "application/x-json", bytes.NewBufferString(payload))
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if (response.StatusCode != 200) {
		return fmt.Errorf("Server error response: %s.", response.Status)
	}

	//fmt.Println("response Status:", response.Status)
	//fmt.Println("response Headers:", response.Header)
	//body, _ := ioutil.ReadAll(response.Body)
	//fmt.Println("response Body:", string(body))

	if err := json.NewDecoder(response.Body).Decode(&decoder); err != nil {
		return errors.New("Unable to parse response body")
	}

	return nil
}
