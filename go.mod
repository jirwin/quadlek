module github.com/jirwin/quadlek

go 1.13

require (
	github.com/Bullpeen/infobot v0.0.0-20191214170509-b4e3681ba4e1
	github.com/boltdb/bolt v1.3.1
	github.com/dghubble/go-twitter v0.0.0-20220608135633-47eb18e5aab5
	github.com/dghubble/oauth1 v0.7.1
	github.com/google/go-github v17.0.0+incompatible
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/schema v1.2.0
	github.com/jirwin/comics v0.0.0-20180408212830-43822d8acb7c
	github.com/jirwin/gifs-quadlek v0.0.0-20220620083533-f84367d2dbab
	github.com/jirwin/xpost-quadlek v0.0.0-20190210050443-319004b8e32d
	github.com/purdyk/tvdb v0.0.0-20170517053125-80f06ad52285
	github.com/satori/go.uuid v1.2.0
	github.com/slack-go/slack v0.11.0
	github.com/urfave/cli v1.22.9
	github.com/zmb3/spotify v1.3.0
	go.uber.org/zap v1.21.0
	golang.org/x/oauth2 v0.0.0-20220608161450-d0670ef3b1eb
	google.golang.org/protobuf v1.28.0
	gopkg.in/olivere/elastic.v5 v5.0.86
)

replace github.com/nlopes/slack => github.com/jirwin/slack v0.6.1-0.20200216211639-aba6477b6931
