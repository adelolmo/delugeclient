package delugeclient_test

import (
	"fmt"
	"github.com/adelolmo/delugeclient"
	"github.com/bmizerany/assert"
	"github.com/bmizerany/pat"
	"github.com/drewolson/testflight"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

func TestNewDelugeNoServerUrl(t *testing.T) {
	assertPanic(t, func() {
		delugeclient.NewDeluge("", "")
	})
}

func TestConnection(t *testing.T) {
	testflight.WithServer(Handler(""), func(r *testflight.Requester) {
		client := delugeclient.NewDeluge("http://"+r.Url(""), "pass")
		fmt.Println(r.Url(""))
		if err := client.Connect(); err != nil {
			fmt.Println(err)
			t.Fail()
		}
	})
}

func TestConnectionWrongPassword(t *testing.T) {
	testflight.WithServer(WrongPasswordHandler(), func(r *testflight.Requester) {
		client := delugeclient.NewDeluge("http://"+r.Url(""), "xxx")
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
			client := delugeclient.NewDeluge("http://"+r.Url(""), "pass")
			if err := client.Connect(); err != nil {
				t.Fail()
			}
			if err := client.AddMagnet("magnet:?xt=urn:btih:asdfgh123456"); err != nil {
				fmt.Println(err)
				t.Fail()
			}
		})
}

func TestGettingNoFiles(t *testing.T) {
	testflight.WithServer(Handler(
		`{
		  "id": 2,
		  "result": {
			"type": "dir",
			"contents": {}
		  },
		  "error": null
		}`), func(r *testflight.Requester) {
		client := delugeclient.NewDeluge("http://"+r.Url(""), "pass")
		if err := client.Connect(); err != nil {
			t.Fail()
		}
		torrent, err := client.Get("id")
		if err != nil {
			fmt.Println(err)
			t.Fail()
		}
		fmt.Println(torrent)
		assert.Equal(t, "id", torrent.Id)
		assert.Equal(t, "", torrent.Name)
		assert.Equal(t, 0.0, torrent.ShareRatio)
		assert.Equal(t, 0, len(torrent.Files))
	})
}

func TestGettingSingleFile(t *testing.T) {
	testflight.WithServer(Handler(
		`{
		  "id": 822,
		  "result": {
		    "type": "dir",
		    "contents": {
		      "Single File.mp4": {
			"priority": 1,
			"index": 0,
			"offset": 0,
			"progress": 85.989601135254,
		    "ratio": 1,
			"path": "Single File.mp4",
			"type": "file",
			"size": 465171004
		      }
		    }
		  },
		  "error": null
		}`),
		func(r *testflight.Requester) {
			client := delugeclient.NewDeluge("http://"+r.Url(""), "pass")
			if err := client.Connect(); err != nil {
				t.Fail()
			}
			torrent, err := client.Get("id")
			if err != nil {
				fmt.Println(err)
				t.Fail()
			}
			fmt.Println(torrent)
			assert.Equal(t, "id", torrent.Id)
			assert.Equal(t, "Single File.mp4", torrent.Name)
			assert.Equal(t, 1.0, torrent.ShareRatio)
			assert.Equal(t, 1, len(torrent.Files))
			assert.Equal(t, "Single File.mp4", torrent.Files[0])
		})
}

func TestGettingMultipleFiles(t *testing.T) {
	testflight.WithServer(Handler(
		`{
		  "id": 1,
		  "result": {
		    "type": "dir",
		    "contents": {
		      "Some.Linux.Distro": {
			"priority": 1,
			"path": "Some.Linux.Distro",
			"progress": 85.989601135254,
			"progresses": [
			  10199684.56,
			  0.3,
			  0.57
			],
		    "ratio": 1,
			"type": "dir",
			"contents": {
			  "README.txt": {
			    "priority": 1,
			    "index": 1,
			    "offset": 1019968456,
			    "progress": 1,
			    "path": "Some.Linux.Distro\/README.txt",
			    "type": "file",
			    "size": 30
			  },
			  "Distribution.iso": {
			    "priority": 1,
			    "index": 0,
			    "offset": 0,
			    "progress": 1,
			    "path": "Some.Linux.Distro\/Distribution.iso",
			    "type": "file",
			    "size": 1019968456
			  },
			  "distribution.nfo": {
			    "priority": 1,
			    "index": 2,
			    "offset": 1019968486,
			    "progress": 1,
			    "path": "Some.Linux.Distro\/distribution.nfo",
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
			client := delugeclient.NewDeluge("http://"+r.Url(""), "pass")
			if err := client.Connect(); err != nil {
				t.Fail()
			}
			torrent, err := client.Get("id")
			if err != nil {
				fmt.Println(err)
				t.Fail()
			}
			fmt.Println(torrent)
			assert.Equal(t, "id", torrent.Id)
			assert.Equal(t, "Some.Linux.Distro", torrent.Name)
			assert.Equal(t, 1.0, torrent.ShareRatio)
			assert.Equal(t, 85.989601135254, torrent.Progress)
			assertContains(t, []string{"README.txt", "Distribution.iso", "distribution.nfo"}, torrent.Files)
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
			"name": "Some.Linux.Distro"
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
			client := delugeclient.NewDeluge("http://"+r.Url(""), "pass")
			if err := client.Connect(); err != nil {
				t.Fail()
			}
			torrents, err := client.GetAll()
			if err != nil {
				fmt.Println(err)
				t.Fail()
			}
			fmt.Println(torrents)
			assert.Equal(t, 2, len(torrents))
			assert.Equal(t, "asdfgh123456", torrents[0].Id)
			assert.Equal(t, "Some.Linux.Distro", torrents[0].Name)
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
			client := delugeclient.NewDeluge("http://"+r.Url(""), "pass")
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
		if strings.Contains(string(body), "auth.login") {
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

func assertContains(t *testing.T, exp []string, got []string) {
	found := 0
	for expItem := range exp {
		for gotItem := range got {
			if expItem == gotItem {
				found++
			}
		}
	}
	if len(exp) != found {
		t.Errorf("%v:%v", exp, got)
	}
}
