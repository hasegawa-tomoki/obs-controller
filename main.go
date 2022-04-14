package main

import (
	"log"
	"obs-controller/obs"
	"obs-controller/schedule"
	"os"
	"os/signal"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/kawasin73/htask/cron"
	"github.com/slack-go/slack"
)

var err = godotenv.Load(`.env`)
var slackToken string = os.Getenv(`SLACK_TOKEN`)
var slackChannel string = `#` + os.Getenv(`SLACK_CHANNEL`)
var obsControllerName string = os.Getenv(`OBS_CONTROLLER_NAME`)

var schedules []schedule.Schedule
var sceneSequences []schedule.SceneSequence

func main() {
	var wg sync.WaitGroup
	cron := cron.NewCron(&wg, cron.Option{
		Workers: 1,
	})
	slack := obs.SlackContainer{
		Client:  slack.New(slackToken),
		Channel: slackChannel,
	}
	obscon := obs.NewObsCon(obsControllerName, slack)
	defer func() {
		cron.Close()
		wg.Wait()
	}()

	donec := make(chan bool)

	go apiStart(cron, obscon, donec)
	go obs.Receive(obscon, donec)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	<-interrupt
}

func apiStart(c *cron.Cron, obscon *obs.ObsCon, donec chan bool) {
	gin.SetMode(os.Getenv(`GIN_MODE`))

	r := gin.Default()
	r.GET("/stats", func(con *gin.Context) {
		stats := obs.GetStats(obscon)

		con.JSON(200, gin.H{
			"status": "OK",
			"data":   stats,
		})
	})
	r.GET("/scenes", func(con *gin.Context) {
		scenes := obs.GetSceneList(obscon)

		con.JSON(200, gin.H{
			"status": "OK",
			"data":   scenes,
		})
	})
	r.GET("/scene/:name", func(con *gin.Context) {
		name := con.Param("name")
		scenes := obs.GetSceneItemList(obscon, name)

		con.JSON(200, gin.H{
			"status": "OK",
			"data":   scenes,
		})
	})
	r.GET("/source-settings/:name", func(con *gin.Context) {
		name := con.Param("name")
		scenes := obs.GetSourceSettings(obscon, name)

		con.JSON(200, gin.H{
			"status": "OK",
			"data":   scenes,
		})
	})
	r.GET("/schedules", func(con *gin.Context) {
		log.Println("GET /schedules")
		log.Println(schedules)

		con.JSON(200, gin.H{
			"status": "OK",
			"data":   schedules,
		})
	})
	r.POST("/schedules", func(con *gin.Context) {
		log.Println("POST /schedules")
		var newSchedules []schedule.Schedule
		if con.ShouldBind(&newSchedules) == nil {
			log.Println("newSchedules: ", newSchedules)
			con.JSON(200, gin.H{
				"status": "OK",
			})
			schedules = schedule.LoadSchedules(c, obscon, newSchedules)
			log.Println("Current jobs: ")
			log.Println("  ", schedules)
		} else {
			log.Println("Bind failed")
			con.JSON(200, gin.H{
				"status": "error",
				"error":  "Invalid request body.",
			})
		}
	})
	r.GET("/scene-sequences", func(con *gin.Context) {
		log.Println("GET /scene-sequences")
		log.Println(sceneSequences)

		con.JSON(200, gin.H{
			"status": "OK",
			"data":   sceneSequences,
		})
	})
	r.POST("/scene-sequences", func(con *gin.Context) {
		log.Println("POST /scene-sequences")
		var newSceneSequences []schedule.SceneSequence
		if con.ShouldBind(&newSceneSequences) == nil {
			log.Println("newSceneSequences: ", newSceneSequences)
			con.JSON(200, gin.H{
				"status": "OK",
			})
			sceneSequences = schedule.LoadSceneSequences(newSceneSequences)
		} else {
			log.Println("Bind failed")
			con.JSON(200, gin.H{
				"status": "error",
				"error":  "Invalid request body.",
			})
		}
	})
	r.Run(":8181")
}
