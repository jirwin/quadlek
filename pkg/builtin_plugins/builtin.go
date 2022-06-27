package builtin_plugins

import (
	"github.com/jirwin/quadlek/pkg/builtin_plugins/admin"
	"github.com/jirwin/quadlek/pkg/builtin_plugins/comics"
	"github.com/jirwin/quadlek/pkg/builtin_plugins/echo"
	"github.com/jirwin/quadlek/pkg/builtin_plugins/eslogs"
	"github.com/jirwin/quadlek/pkg/builtin_plugins/gifs"
	"github.com/jirwin/quadlek/pkg/builtin_plugins/github"
	"github.com/jirwin/quadlek/pkg/builtin_plugins/karma"
	"github.com/jirwin/quadlek/pkg/builtin_plugins/nextep"
	"github.com/jirwin/quadlek/pkg/builtin_plugins/random"
	"github.com/jirwin/quadlek/pkg/builtin_plugins/spotify"
	"github.com/jirwin/quadlek/pkg/builtin_plugins/twitter"
)

var (
	Admin             = admin.Register
	AdminInteractions = admin.RegisterInteraction
	Comics            = comics.Register
	Echo              = echo.Register
	EsLogs            = eslogs.Register
	Gifs              = gifs.Register
	Github            = github.Register
	Karma             = karma.Register
	NextEp            = nextep.Register
	Random            = random.Register
	Spotify           = spotify.Register
	Twitter           = twitter.Register
)
