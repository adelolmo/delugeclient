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
	Code    int    `json:"code"`
}

type RpcResponse struct {
	Id     int      `json:"id"`
	Result bool     `json:"result"`
	Error  RpcError `json:"error"`
}

func (r RpcResponse) String() string {
	return fmt.Sprintf("id: '%d' result: '%s' error: {%s}", r.Id, r.Result, r.Error)
}
func (e RpcError) String() string {
	return fmt.Sprintf("code: '%d' message: '%s'", e.Code, e.Message)
}

type Torrent struct {
	Id         string
	Name       string
	Progress   float64
	ShareRatio float64
	Files      []string
}

func (t *Torrent) String() string {
	return fmt.Sprintf("id=%s name=%s ratio=%f files=%s", t.Id, t.Name, t.ShareRatio, t.Files)
}

type TorrentEntry struct {
	Message  string  `json:"message"`
	Progress float64 `json:"progress"`
	Ratio    float64 `json:"ratio"`
	Name     string  `json:"name"`
}

type TorrentSet struct {
	Map map[string]TorrentEntry `json:"torrents"`
}

type AllResponse struct {
	Index    int        `json:"id"`
	Torrents TorrentSet `json:"result"`
	Error    RpcError   `json:"error"`
}

// Initialize client
func NewDeluge(serverUrl, password string) *Deluge {
	if len(serverUrl) == 0 {
		panic("serverUrl cannot be empty")
	}
	if len(password) == 0 {
		panic("password cannot be empty")
	}
	log.SetOutput(os.Stdout)
	options := cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	}
	cookieJar, err := cookiejar.New(&options)
	if err != nil {
		log.Fatal(err)
	}
	config := &tls.Config{InsecureSkipVerify: true}
	tr := &http.Transport{TLSClientConfig: config}
	return &Deluge{
		ServiceUrl: serverUrl + "/json",
		Password:   password,
		Index:      1,
		HttpClient: http.Client{Jar: cookieJar, Transport: tr},
	}
}

// Establish connection to the server
func (d *Deluge) Connect() error {
	var payload = fmt.Sprintf(
		`{"id":%d, "method":"auth.login", "params":["%s"]}`,
		d.Index, d.Password)
	var rr RpcResponse
	err := sendRequest(d.HttpClient, d.ServiceUrl, payload, &rr)

	if err != nil {
		return err
	}
	if !rr.Result {
		return fmt.Errorf("error code %d! %s", rr.Error.Code, rr.Error.Message)
	}

	d.Index ++
	return nil
}

// Adds a magnet/torrent link
func (d *Deluge) AddMagnet(magnet string) error {
	var payload = fmt.Sprintf(
		`{"id":%d, "method":"web.add_torrents", "params":[[{"path":"%s", "options":""}]]}`,
		d.Index, magnet)
	var rr RpcResponse
	err := sendRequest(d.HttpClient, d.ServiceUrl, payload, &rr)

	if err != nil {
		return err
	}
	if rr.Error.Code > 0 {
		log.Println(rr)
		return fmt.Errorf("error code %d! %s", rr.Error.Code, rr.Error.Message)
	}
	d.Index ++
	return nil
}

// Moves torrent to the queue top
func (d *Deluge) MoveToQueueTop(torrentId string) error {
	var payload = fmt.Sprintf(
		`{"id":%d, "method":"core.queue_top", "params":[["%s"]]}`,
		d.Index, torrentId)
	var rr RpcResponse
	err := sendRequest(d.HttpClient, d.ServiceUrl, payload, &rr)

	if err != nil {
		return err
	}
	if rr.Error.Code > 0 {
		log.Println(rr)
		return fmt.Errorf("error code %d! %s", rr.Error.Code, rr.Error.Message)
	}
	d.Index ++
	return nil
}

type TorrentResult struct {
	Type     string                   `json:"type"`
	Contents map[string]TorrentDetail `json:"contents"`
}

type TorrentContent struct {
	Index         int           `json:"id"`
	TorrentResult TorrentResult `json:"result"`
	Error         RpcError      `json:"error"`
}

type TorrentDetail struct {
	Priority        int64                    `json:"priority"`
	Path            string                   `json:"path"`
	Type            string                   `json:"type"`
	ShareRatio      float64                  `json:"ratio"`
	Progress        float64                  `json:"progress"`
	TorrentEntryMap map[string]TorrentDetail `json:"contents"`
}

type Detail struct {
	Path string `json:"path"`
}

// Gets the link details about a single link given its hash id (torrentId)
func (d *Deluge) Get(torrentId string) (*Torrent, error) {
	var payload = fmt.Sprintf(
		`{"id":%d, "method":"web.get_torrent_files", "params":["%s"]}`,
		d.Index, torrentId)
	var rr TorrentContent
	err := sendRequest(d.HttpClient, d.ServiceUrl, payload, &rr)
	if err != nil {
		panic(err)
	}
	if rr.Error.Code > 0 {
		return nil, fmt.Errorf("error code %d! %s", rr.Error.Code, rr.Error.Message)
	}

	if rr.TorrentResult.Type != "dir" {
		return nil, nil
	}

	for k, v := range rr.TorrentResult.Contents {

		contents := rr.TorrentResult.Contents[k]
		if len(contents.TorrentEntryMap) == 0 {
			files := make([]string, 0, 1)
			return &Torrent{
				Id:         torrentId,
				Name:       contents.Path,
				Files:      append(files, contents.Path),
				ShareRatio: contents.ShareRatio,
			}, nil
		}

		files := make([]string, 0, len(contents.TorrentEntryMap))
		for x, y := range contents.TorrentEntryMap {

			if y.Type == "file" {
				//fmt.Println("type: ", y.Type, " key: ", x)
				files = append(files, x)
			}
		}
		d.Index ++
		return &Torrent{
			Id:         torrentId,
			Name:       v.Path,
			Files:      files,
			ShareRatio: v.ShareRatio,
			Progress:   v.Progress,
		}, nil
	}
	return nil, nil
}

// Gets the link details off all entries
func (d *Deluge) GetAll() ([]Torrent, error) {
	var payload = fmt.Sprintf(
		`{"id":%d, "method":"web.update_ui", "params":[["name", "ratio", "message", "progress"],{}]}`,
		d.Index)
	var rr AllResponse
	err := sendRequest(d.HttpClient, d.ServiceUrl, payload, &rr)
	if err != nil {
		panic(err)
	}
	if rr.Error.Code > 0 {
		log.Println(rr)
		return nil, fmt.Errorf("error code %d! %s", rr.Error.Code, rr.Error.Message)
	}

	torrents := make([]Torrent, 0, len(rr.Torrents.Map))
	for k, v := range rr.Torrents.Map {
		torrents = append(torrents, Torrent{Id: k, Name: v.Name, ShareRatio: v.Ratio, Progress: v.Progress})
	}
	d.Index ++
	return torrents, nil
}

// Removes a link given its hash id (torrentId)
func (d *Deluge) Remove(torrentId string) error {
	var payload = fmt.Sprintf(
		`{"id":%d, "method":"core.remove_torrent", "params":["%s",true]}`,
		d.Index, torrentId)
	var rr RpcResponse
	err := sendRequest(d.HttpClient, d.ServiceUrl, payload, &rr)
	if err != nil {
		return err
	}

	if rr.Error.Code > 0 {
		log.Println(rr)
		return fmt.Errorf("error code %d! %s", rr.Error.Code, rr.Error.Message)
	}
	d.Index ++
	return nil
}

func sendRequest(httpClient http.Client, url, payload string, decoder interface{}) error {
	response, err := httpClient.Post(url, "application/json", bytes.NewBufferString(payload))
	if err != nil {
		return fmt.Errorf("connection error. %s", err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return fmt.Errorf("server error response: %s", response.Status)
	}

	//fmt.Println("response Status:", response.Status)
	//fmt.Println("response Headers:", response.Header)
	//body, _ := ioutil.ReadAll(response.Body)
	//fmt.Println("response Body:", string(body))

	if err := json.NewDecoder(response.Body).Decode(&decoder); err != nil {
		return errors.New("unable to parse response body")
	}

	return nil
}
