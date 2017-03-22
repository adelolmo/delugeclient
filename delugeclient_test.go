package delugeclient_test

import (
	"github.com/adelolmo/delugeclient"
	"testing"
	"fmt"
	"net/http"
	"io"
	"io/ioutil"
	"strings"
	"github.com/drewolson/testflight"
	"github.com/bmizerany/pat"
	"github.com/bmizerany/assert"
)

func TestNewDelugeNoServerUrl(t *testing.T) {
	assertPanic(t, func() {
		delugeclient.NewDeluge("", "")
	})
}
func TestConnection(t *testing.T) {
	testflight.WithServer(Handler(""), func(r *testflight.Requester) {
		client := delugeclient.NewDeluge("http://" + r.Url(""), "pass")
		fmt.Println(r.Url(""))
		if err := client.Connect(); err != nil {
			fmt.Println(err)
			t.Fail()
		}
	})
}

func TestConnectionWrongPassword(t *testing.T) {
	testflight.WithServer(WrongPasswordHandler(), func(r *testflight.Requester) {
		client := delugeclient.NewDeluge("http://" + r.Url(""), "xxx")
		if err := client.Connect(); err == nil {
			t.Fail()
		}
	})
}

func TestAddingMagnet(t *testing.T) {
	testflight.WithServer(Handler(
		`
		{"id": 2, "result": true, "error":{"code":0, "message":""}}
		`),
		func(r *testflight.Requester) {
			client := delugeclient.NewDeluge("http://" + r.Url(""), "pass")
			if err := client.Connect(); err != nil {
				t.Fail()
			}
			if err := client.AddMagnet("magnet:?xt=urn:btih:asdfgh123456"); err != nil {
				fmt.Println(err)
				t.Fail()
			}
		})
}

func TestGetting(t *testing.T) {
	testflight.WithServer(Handler(
		`{
		  "id": 1,
		  "result": {
		    "type": "dir",
		    "contents": {
		      "Some.Linux.Disto": {
			"priority": 1,
			"path": "Some.Linux.Disto",
			"progress": 1,
			"progresses": [
			  10199684.56,
			  0.3,
			  0.57
			],
			"type": "dir",
			"contents": {
			  "README.txt": {
			    "priority": 1,
			    "index": 1,
			    "offset": 1019968456,
			    "progress": 1,
			    "path": "Some.Linux.Disto\/README.txt",
			    "type": "file",
			    "size": 30
			  },
			  "Distribution.iso": {
			    "priority": 1,
			    "index": 0,
			    "offset": 0,
			    "progress": 1,
			    "path": "Some.Linux.Disto\/Distribution.iso",
			    "type": "file",
			    "size": 1019968456
			  },
			  "distribution.nfo": {
			    "priority": 1,
			    "index": 2,
			    "offset": 1019968486,
			    "progress": 1,
			    "path": "Some.Linux.Disto\/distribution.nfo",
			    "type": "file",
			    "size": 57
			  }
			},
			"size": 1019968543
		      }
		    }
		  },
		  "error": null
		}`),
		func(r *testflight.Requester) {
			client := delugeclient.NewDeluge("http://" + r.Url(""), "pass")
			if err := client.Connect(); err != nil {
				t.Fail()
			}
			torrent, err := client.Get("id")
			if (err != nil) {
				fmt.Println(err)
				t.Fail()
			}
			fmt.Println(torrent)
			assert.Equal(t, "id", torrent.Id)
			assert.Equal(t, "Some.Linux.Disto", torrent.Name)
			assert.Equal(t, 1.0, torrent.ShareRatio)
			assert.Equal(t, "README.txt", torrent.Files[0])
			assert.Equal(t, "Distribution.iso", torrent.Files[1])
			assert.Equal(t, "distribution.nfo", torrent.Files[2])
		})
}

func TestGettingAll(t *testing.T) {
	testflight.WithServer(Handler(
		`
		{
		  "id": 2,
		  "result": {
		    "torrents": {
		      "asdfgh123456": {
			"message": "OK",
			"ratio": 4.08238410949707,
			"name": "Some.Linux.Disto"
		      },
		      "123456asdfgh": {
			"message": "OK",
			"ratio": 0.0008267719531431794,
			"name": "Some.Video"
		      }
		    }
		  },
		  "error": null
		}
		`),
		func(r *testflight.Requester) {
			client := delugeclient.NewDeluge("http://" + r.Url(""), "pass")
			if err := client.Connect(); err != nil {
				t.Fail()
			}
			torrents, err := client.GetAll()
			if (err != nil) {
				fmt.Println(err)
				t.Fail()
			}
			fmt.Println(torrents)
			assert.Equal(t, 2, len(torrents))
			assert.Equal(t, "asdfgh123456", torrents[0].Id)
			assert.Equal(t, "Some.Linux.Disto", torrents[0].Name)
			assert.Equal(t, 4.08238410949707, torrents[0].ShareRatio)
			assert.Equal(t, "123456asdfgh", torrents[1].Id)
			assert.Equal(t, "Some.Video", torrents[1].Name)
			assert.Equal(t, 0.0008267719531431794, torrents[1].ShareRatio)
		})
}

func TestRemovingTorrent(t *testing.T) {
	testflight.WithServer(Handler(
		`
		{"id": 2, "result": true, "error":{"code":0, "message":""}}
		`),
		func(r *testflight.Requester) {
			client := delugeclient.NewDeluge("http://" + r.Url(""), "pass")
			if err := client.Connect(); err != nil {
				t.Fail()
			}
			if err := client.Remove("asdfgh123456"); err != nil {
				fmt.Println(err)
				t.Fail()
			}
		})
}

func WrongPasswordHandler() http.Handler {
	m := pat.New()
	m.Post("/json", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(200)
		io.WriteString(w, `{"id": 2, "result": false, "error": null}`)
	}))
	return m
}
func Handler(response string) http.Handler {
	m := pat.New()
	m.Post("/json", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		body, _ := ioutil.ReadAll(req.Body)
		w.WriteHeader(200)
		if (strings.Contains(string(body), "auth.login")) {
			io.WriteString(w, `{"id": 2, "result": true, "error": null}`)
			return
		}
		io.WriteString(w, response)

	}))
	return m
}

func assertPanic(t *testing.T, f func()) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	f()
}
