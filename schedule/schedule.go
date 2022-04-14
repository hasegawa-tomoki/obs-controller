package schedule

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"obs-controller/obs"
	"time"

	"github.com/kawasin73/htask/cron"
)

const prepareBeforeSec = 10

var cancelSchedules []func()
var cancelSequences []func()

var sceneSequences []SceneSequence

type Scene struct {
	Name            string `json:"name"`
	MediaSourceName string `json:"media_source_name"`
	MediaSourcePath string `json:"media_source_path"`
	SceneLengthMsec int    `json:"scene_length_msec"`
}

type Schedule struct {
	SwitchAt              time.Time `json:"switch_at" binding:"required"` // RFC3339
	Scene                 Scene     `json:"scene"`
	NextSceneSequenceUuid string    `json:"next_scene_sequence_uuid"`
}

type SceneSequence struct {
	Uuid   string  `json:"uuid"`
	Name   string  `json:"name"`
	Scenes []Scene `json:"scenes"`
}

func FromJson(str string) (Schedule, error) {
	var s Schedule
	if err := json.Unmarshal([]byte(str), &s); err != nil {
		log.Println("Bind error!!!")
		log.Println(err)
		return s, err
	}
	// log.Println(s)
	return s, nil
}

func sequenceTask(c *cron.Cron, obscon *obs.ObsCon, ssq SceneSequence, idx int, first bool) {
	if idx >= len(ssq.Scenes) {
		log.Println(`Invalid index for `, ssq.Name, `of index %d`, idx)
		return
	}
	scene := ssq.Scenes[idx]

	// Switch scene
	log.Println(`* sequenceTask for `, ssq.Name, `of index`, idx)
	if first {
		obscon.Slack.Send2slack(fmt.Sprintf(`:information_source: [%s] シーンシーケンス "%s" に切り替えます。`, obscon.Name, ssq.Name))
	}

	obs.SwitchScene(obscon, scene.Name)

	nextTime := time.Now().Add(time.Duration(scene.SceneLengthMsec) * time.Millisecond)
	newIdx := (idx + 1) % len(ssq.Scenes)
	nextScene := ssq.Scenes[newIdx]
	// Prepare
	cp, _ := c.Once(nextTime.Add(-1 * prepareBeforeSec * time.Second)).Run(func() {
		prepareTask(c, obscon, nextScene.Name, obs.Source{Name: nextScene.MediaSourceName, Path: nextScene.MediaSourcePath})
	})
	cancelSequences = append(cancelSequences, cp)
	// Switch scene
	cs, _ := c.Once(nextTime).Run(func() { sequenceTask(c, obscon, ssq, newIdx, false) })
	cancelSequences = append(cancelSequences, cs)
}

func prepareTask(c *cron.Cron, obscon *obs.ObsCon, sceneName string, _source obs.Source) {
	source := obs.Source{
		Name: _source.Name,
		Path: _source.Path,
	}

	log.Println(`* prepareTask for scene`, sceneName, `source.Path:`, source.Path)
	ok := obs.PrepareScene(obscon, sceneName, source)
	if !ok {
		obscon.Slack.Send2slack(fmt.Sprintf(`:bangbang: [%s] シーン "%s" の準備に失敗しました。スケジュールに影響がないか確認してください。`, obscon.Name, sceneName))
	}
}

func scheduledSwitchTask(c *cron.Cron, obscon *obs.ObsCon, s Schedule) {
	log.Println(`* scheduledSwitchTask for scene`, s.Scene.Name)
	obscon.Slack.Send2slack(fmt.Sprintf(`:information_source: [%s] シーン "%s" に切り替えます。（%s 開始）`, obscon.Name, s.Scene.Name, s.SwitchAt))

	cancelCrons(cancelSequences)
	log.Println(`    file: `, s.Scene.MediaSourcePath)
	obs.SwitchScene(obscon, s.Scene.Name)

	// Next scene sequence
	if s.NextSceneSequenceUuid != "" {
		log.Println("Next: scene sequence index 0")
		ssq, err := findSceneSequenceWithUuid(s.NextSceneSequenceUuid)
		if err == nil {
			if len(ssq.Scenes) > 0 {
				nextTime := s.SwitchAt.Add(time.Duration(s.Scene.SceneLengthMsec) * time.Millisecond)
				// Prepare
				cp, _ := c.Once(nextTime.Add(-1 * prepareBeforeSec * time.Second)).Run(func() {
					prepareTask(c, obscon, ssq.Scenes[0].Name, obs.Source{
						Name: fmt.Sprintf("%s", ssq.Scenes[0].MediaSourceName),
						Path: fmt.Sprintf("%s", ssq.Scenes[0].MediaSourcePath),
					})
				})
				cancelSequences = append(cancelSequences, cp)
				// Switch scene
				cs, _ := c.Once(nextTime).Run(func() {
					sequenceTask(c, obscon, ssq, 0, true)
				})
				cancelSequences = append(cancelSequences, cs)

				// obscon.Slack.Send2slack(fmt.Sprintf(`:information_source: トーク終了時刻 %s にシーンシーケンス "%s" に切り替えます。`, nextTime, ssq.Name))
			} else {
				obscon.Slack.Send2slack(fmt.Sprintf(`:information_source: [%s] シーンシーケンス "%s" にシーンが無いので切替は発生しません。`, obscon.Name, ssq.Name))
			}
		} else {
			obscon.Slack.Send2slack(fmt.Sprintf(`:bangbang: [%s] 指定されたシーンシーケンスが見つかりません。（UUID: %s）`, obscon.Name, s.NextSceneSequenceUuid))
		}
	} else {
		obscon.Slack.Send2slack(fmt.Sprintf(`:information_source: [%s] 終了後のシーン切替はありません。`, obscon.Name))
	}
}

func LoadSchedules(c *cron.Cron, obscon *obs.ObsCon, newSchedules []Schedule) []Schedule {
	cancelCrons(cancelSchedules)
	cancelCrons(cancelSequences)

	schedules := []Schedule{}
	for _, s := range newSchedules {
		if s.SwitchAt.After(time.Now()) {
			_s := Schedule{
				SwitchAt: s.SwitchAt,
				Scene: Scene{
					Name:            s.Scene.Name,
					MediaSourceName: s.Scene.MediaSourceName,
					MediaSourcePath: s.Scene.MediaSourcePath,
					SceneLengthMsec: s.Scene.SceneLengthMsec,
				},
				NextSceneSequenceUuid: s.NextSceneSequenceUuid,
			}
			// Prepare
			log.Println(`Prepare task at`, s.SwitchAt.Add(-1*prepareBeforeSec*time.Second), `for`, s.SwitchAt)
			prepareCancel, _ := c.Once(s.SwitchAt.Add(-1 * prepareBeforeSec * time.Second)).Run(func() {
				prepareTask(c, obscon, _s.Scene.Name, obs.Source{
					Name: _s.Scene.MediaSourceName,
					Path: _s.Scene.MediaSourcePath,
				})
			})
			cancelSchedules = append(cancelSchedules, prepareCancel)
			// Switch scene
			switchCancel, _ := c.Once(s.SwitchAt).Run(func() {
				log.Println(_s)
				scheduledSwitchTask(c, obscon, _s)
			})
			cancelSchedules = append(cancelSchedules, switchCancel)

			schedules = append(schedules, _s)
		}
	}
	return schedules
}

func LoadSceneSequences(newSceneSequences []SceneSequence) []SceneSequence {
	sceneSequences = newSceneSequences
	return sceneSequences
}

func cancelCrons(cancels []func()) {
	for _, cancel := range cancels {
		cancel()
	}
}

func findSceneSequenceWithUuid(uuid string) (SceneSequence, error) {
	for _, s := range sceneSequences {
		if s.Uuid == uuid {
			return s, nil
		}
	}
	return SceneSequence{}, errors.New("Not found")
}
