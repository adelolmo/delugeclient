# delugeclient
Simple Deluge client for Go

## Features

* Add mangnet link
* Get list of all torrents in the server
* Remove a torrent

## Usage

```go
    import "github.com/adelolmo/delugeclient"

    // Connect to Deluge server
    deluge := delugeclient.NewDeluge("deluge_server_url", "deluge_password")
    if err := deluge.Connect(); err != nil {
        panic(err)
    }

    // Add a magnet link
    if err := deluge.AddMagnet("magnet:?xt=urn:btih:032f37e3b98f60148a6..."); err != nil {
        panic(err)
    }

    torrents, err := deluge.GetAll()
    if err != nil {
        panic(err)
    }
    for _, t := range torrents {
        fmt.Printf("%s,%s,%f,%f\n", t.Id, t.Name, t.Progress, t.ShareRatio)
    }

    if err = deluge.Remove(t.Id); err != nil {
        panic(err)
    }
```