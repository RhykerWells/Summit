package run

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/RhykerWells/Summit/bot"
	"github.com/RhykerWells/Summit/commands/economy/models"
	"github.com/RhykerWells/Summit/common"
	"github.com/RhykerWells/Summit/web"
	log "github.com/sirupsen/logrus"
)

// Init initialises the core database system, discord gateway connection, and
// starts all the additional bot services
func Init() {
	testing := flag.Bool("testing", false, "Enable test mode")
	flag.Parse()

	common.ConfigTestMode = *testing

	err := common.Init()
	if err != nil {
		log.WithError(err).Fatal("Failed to start core")
	}

	bot.Run(common.Session, common.PQ)
	common.Run(common.Session)
	web.Run()
}

// Run enables the shutdown services to safely stop and close the bot
func Run() {
	shutdown()
}

// shutdown safely stops the bot and runs the required cleanup functions
func shutdown() {
	sc := make(chan os.Signal, 2)
	signal.Notify(sc, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	log.Infoln("Exiting now....")
	shutdownCleanup()
	os.Exit(0)
}

// shutdownCleanup runs the cleanup functions for the bot shutdown
func shutdownCleanup() {
	log.Warnln("Running cleanup functions")
	ok, err := models.EconomyCreateitems().DeleteAllG(context.Background())
	if err != nil {
		log.Errorln("Error running cleanup for economy:", ok)
	}
}
