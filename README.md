# devo

##Simple Query
```
package main

import (
        "log"
        "time"
        "github.com/GolangResources/devo/v1"
)

const (
                LAYOUT = "2006-01-02 15:04:05"
)

func main() {
        dateFrom, _ := time.Parse(LAYOUT, "2020-02-10 15:30:00")
        dateTo, _ := time.Parse(LAYOUT, "2020-02-10 18:36:00")
        devoConf := devo.DevoClient{
                APIKey: "",
                APISecret: "",
                SerreaURL: "https://serreaurl.devo.com/v2/search/query",
                Debug: false,
                BufferSize: 4096,
        }
        d := devo.Init(&devoConf)
        resultmsg := make(chan string, 4096)
        go d.QueryRaw(dateFrom.Unix(), dateTo.Unix(), "from app.apache.access select * where message -> \"favicon.ico\"", resultmsg)
        for msg := range resultmsg {
                log.Println(msg)
        }

```

##Continuous Query
```
package main

import (
        "log"
        "time"
        "github.com/GolangResources/devo/v1"
)

const (
                LAYOUT = "2006-01-02 15:04:05"
)

func main() {
        dateFrom, _ := time.Parse(LAYOUT, "2020-02-10 15:30:00")
        devoConf := devo.DevoClient{
                APIKey: "",
                APISecret: "",
                SerreaURL: "https://serreaurl.devo.com/v2/search/query",
                Debug: false,
                BufferSize: 4096,
        }
        d := devo.Init(&devoConf)
        resultmsg := make(chan string, 4096)
        go d.ContinuousQuery(dateFrom.Unix(), "from app.apache.access select * where message -> \"favicon.ico\"", resultmsg)
        for msg := range resultmsg {
                log.Println(msg)
        }

```
