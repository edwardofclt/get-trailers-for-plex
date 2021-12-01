package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gosimple/slug"
	"github.com/kkdai/youtube/v2"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Radarr           *RadarrConfig
	DownloadLocation string
	SleepTime        time.Duration
}

type RadarrConfig struct {
	URL string
	Key string
}

var config *Config

func init() {
	sleepTime := os.Getenv("SLEEP_TIME")
	if sleepTime == "" {
		sleepTime = "1440s"
	}
	sleepTimeDuration, err := time.ParseDuration(sleepTime)
	if err != nil {
		logrus.Error(err)
	}

	config = &Config{
		Radarr: &RadarrConfig{
			URL: os.Getenv("RADARR_URL"),
			Key: os.Getenv("RADARR_KEY"),
		},
		DownloadLocation: os.Getenv("DOWNLOAD_LOCATION"),
		SleepTime:        sleepTimeDuration,
	}
}

func main() {
	logrus.Info("Hello, Captain.")
	for {
		movies := fetchMoviesFromRadarr()
		limit := 10
		current := 0
		for _, movie := range movies {
			if limit == current {
				continue
			}

			if movie.Downloaded {
				continue
			}

			logrus.WithFields(logrus.Fields{
				"Title": movie.Title,
			}).Info("movie found")
			if err := downloadTrailer(movie); err != nil {
				logrus.WithError(err).Warn("failed to download movie trailer")
			}
			current++
		}
		logrus.Info("see you next time")
		time.Sleep(config.SleepTime)
	}
}

type Movie struct {
	Title            string `json:"title"`
	TrailerYoutubeID string `json:"youTubeTrailerId"`
	Downloaded       bool   `json:"downloaded"`
}

func downloadTrailer(movie Movie) error {
	yt := youtube.Client{}
	video, err := yt.GetVideo(movie.TrailerYoutubeID)
	if err != nil {
		return err
	}

	filePath := fmt.Sprintf("%s/%s.mp4", config.DownloadLocation, slug.Make(movie.Title))

	if _, err := os.Stat(filePath); err == nil {
		return nil
	}

	formats := video.Formats.WithAudioChannels() // only get videos with audio
	stream, _, err := yt.GetStream(video, &formats[1])
	if err != nil {
		panic(err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	_, err = io.Copy(file, stream)
	if err != nil {
		panic(err)
	}

	return nil
}

func fetchMoviesFromRadarr() []Movie {
	url := fmt.Sprintf("%s/api/v3/movie?apikey=%s", config.Radarr.URL, config.Radarr.Key)
	out, err := http.Get(url)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"URL": config.Radarr.URL,
			"KEY": config.Radarr.Key,
		}).WithError(err).Warn("failed to get list of movies from radarr")
	}

	body, err := ioutil.ReadAll(out.Body)
	if err != nil {
		log.Fatalln(err)
	}

	if out.StatusCode != http.StatusOK {
		logrus.WithFields(logrus.Fields{
			"Body":   string(body),
			"Status": out.StatusCode,
			"URL":    url,
		}).Warn("did not receive status 200")
	}

	var movies []Movie

	err = json.Unmarshal(body, &movies)
	if err != nil {
		logrus.WithError(err).Warn("failed to unmarshal list of movies")
	}

	return movies
}
