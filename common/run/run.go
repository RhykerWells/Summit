package run

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/RhykerWells/Summit/bot"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/Summit/web"
	"github.com/sirupsen/logrus"
)

// Init initialises the core database system, discord gateway connection, and
// starts all the additional bot services
func Init() {
	testing := flag.Bool("testing", false, "Enable test mode")
	flag.Parse()

	common.ConfigTestMode = *testing

	logrus.Infof("Starting Summit version %s", common.VERSION)

	err := common.Init()
	if err != nil {
		logrus.WithError(err).Fatal("Failed to start core")
	}
}

// Run starts the services after initialisation and enables the shutdown services to safely stop and close the bot
func Run() {
	go web.Run()
	bot.Run()
	shutdown()
}

// shutdown safely stops the bot and runs the required cleanup functions
func shutdown() {
	sc := make(chan os.Signal, 2)
	signal.Notify(sc, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	logrus.Infoln("Exiting now....")
	os.Exit(0)
}
