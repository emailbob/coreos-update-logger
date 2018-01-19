package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/go-ini/ini"
	"github.com/urfave/cli"
	elastic "gopkg.in/olivere/elastic.v5"
)

const (
	docType      = "log"
	indexMapping = `{
						"mappings" : {
							"log" : {
								"properties" : {
									"host" : { "type" : "string", "index" : "not_analyzed" },
									"environ" : { "type" : "string", "index" : "not_analyzed" },
									"name" : { "type" : "string", "index" : "not_analyzed" },
									"id" : { "type" : "string", "index" : "not_analyzed" },
									"version" : { "type" : "string", "index" : "not_analyzed" },
									"version_id" : { "type" : "string", "index" : "not_analyzed" },
									"build_id" : { "type" : "string", "index" : "not_analyzed" },
									"pretty_name" : { "type" : "string", "index" : "analyzed" },
									"group" : { "type" : "string", "index" : "not_analyzed" },
									"reboot_strategy" : { "type" : "string", "index" : "not_analyzed" },
									"reboot_window_start" : { "type" : "string", "index" : "not_analyzed" },
									"reboot_window_lenght" : { "type" : "string", "index" : "not_analyzed" },
									"uptime_seconds" : { "type" : "integer", "index" : "not_analyzed" },
									"uptime_minutes" : { "type" : "integer", "index" : "not_analyzed" },
									"uptime_days" : { "type" : "integer", "index" : "not_analyzed" },
									"uptime_seconds" : { "type" : "integer", "index" : "not_analyzed" },
									"time" : { "type" : "date" }
								}
							}
						}
					}`
)

// LockSmithConf type that contains the reboot window start and lenght
type LockSmithConf struct {
	RebootWinStart  string
	RebootWinLenght string
}

// Uptime type that contains the uptime duration of the CoreOS Host
type Uptime struct {
	Seconds int
	Minutes int
	Hours   int
	Days    int
}

// CoreOSRelease type that is sent to elasticsearch in JSON
type CoreOSRelease struct {
	Host            string    `json:"host"`
	Name            string    `json:"name"`
	Environ         string    `json:"environ"`
	ID              string    `json:"id"`
	Version         string    `json:"version"`
	VersionID       string    `json:"version_id"`
	BuildID         string    `json:"build_id"`
	PrettyName      string    `json:"pretty_name"`
	Group           string    `json:"group"`
	RebootStrategy  string    `json:"reboot_strategy"`
	RebootWinStart  string    `json:"reboot_window_start"`
	RebootWinLenght string    `json:"reboot_window_lenght"`
	UptimeSeconds   int       `json:"uptime_seconds"`
	UptimeMinutes   int       `json:"uptime_minutes"`
	UptimeHours     int       `json:"uptime_hours"`
	UptimeDays      int       `json:"uptime_days"`
	Time            time.Time `json:"time"`
}

func writeToES(client *elastic.Client, c *cli.Context, cfg *ini.File) error {
	// get locksmith settings
	reboot, err := getLockSmithConf(c)
	if err != nil {
		log.Println(err)
	}

	// get uptime values
	uptime, err := getUptime(c)
	if err != nil {
		log.Println(err)
	}

	// reload coreos host files
	cfg.Reload()

	t := time.Now()

	data := CoreOSRelease{
		Host:            c.String("host"),
		Environ:         c.String("env"),
		Name:            cfg.Section("").Key("NAME").String(),
		ID:              cfg.Section("").Key("ID").String(),
		Version:         cfg.Section("").Key("VERSION").String(),
		VersionID:       cfg.Section("").Key("VERSION_ID").String(),
		BuildID:         cfg.Section("").Key("BUILD_ID").String(),
		PrettyName:      cfg.Section("").Key("PRETTY_NAME").String(),
		Group:           cfg.Section("").Key("GROUP").String(),
		RebootStrategy:  cfg.Section("").Key("REBOOT_STRATEGY").String(),
		RebootWinLenght: reboot.RebootWinLenght,
		RebootWinStart:  reboot.RebootWinStart,
		UptimeSeconds:   uptime.Seconds,
		UptimeMinutes:   uptime.Minutes,
		UptimeHours:     uptime.Hours,
		UptimeDays:      uptime.Days,
		Time:            t,
	}

	//debug
	//fmt.Printf("%s | %s | %s | %s | %s | %s | %s | %v\n", data.Name, data.ID, data.Version, data.VersionID, data.BuildID, data.Group, data.RebootWinStart, data.UptimeHours)

	// create new elastic search index per day
	indexName := c.String("indexname") + t.Format("-2006-01-02")

	// Create a context
	ctx := context.Background()

	// client, err := elastic.NewClient(elastic.SetURL(c.String("url")))
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// Use the IndexExists service to check if a specified index exists.
	exists, err := client.IndexExists(indexName).Do(ctx)
	if err != nil {
		log.Println(err)
	}
	if !exists {
		// Create an index
		_, err = client.CreateIndex(indexName).
			Body(indexMapping).
			Do(ctx)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Creating index", color.BlueString(indexName))
	}

	// Add a document to the index
	_, err = client.Index().
		Index(indexName).
		Type(docType).
		BodyJson(data).
		Refresh("true").
		Do(ctx)
	if err != nil {
		log.Println(err)
	}

	log.Println("Writing to ElasticSearch at", c.String("url"), "in index", color.BlueString(indexName))

	// Flush to make sure the documents got written.
	_, err = client.Flush().Index(indexName).Do(ctx)
	if err != nil {
		log.Println(err)
	}
	return err
}

// get uptime from host /proc/uptime
func getUptime(c *cli.Context) (Uptime, error) {
	var uptime Uptime
	uptimeString, err := ioutil.ReadFile(c.String("uptime")) // /proc/uptime
	if err != nil {
		log.Println(err)
	}

	secString := strings.Split(string(uptimeString), ".")
	// convert string in seconds to int
	sec, err := strconv.Atoi(secString[0])
	if err != nil {
		log.Println(err)
	}

	uptime.Seconds = sec
	uptime.Minutes = sec / 60
	uptime.Hours = sec / 3600
	uptime.Days = sec / 86400

	return uptime, err
}

func getLockSmithConf(c *cli.Context) (LockSmithConf, error) {
	var rebootWin LockSmithConf

	fi, err := os.Stat(c.String("lock_smith"))
	if err != nil {
		fmt.Println(err)
	}

	if !fi.IsDir() {
		// open a file
		if file, err := os.Open(c.String("lock_smith")); err == nil {

			// make sure file is closed
			defer file.Close()

			// create a new scanner and read the file line by line
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := strings.Split(scanner.Text(), "=")
				// skip first line
				if strings.Index(line[0], "[Service]") != 0 {
					if strings.Index(line[1][1:], "REBOOT_WINDOW_START") == 0 {
						rebootWin.RebootWinStart = line[2][:len(line[2])-1]
					}
					if strings.Index(line[1][1:], "REBOOT_WINDOW_LENGTH") == 0 {
						rebootWin.RebootWinLenght = line[2][:len(line[2])-1]
					}
				}
			}

			// check for errors
			if err = scanner.Err(); err != nil {
				log.Println(err)
			}

		} else {
			log.Println(err)
		}

	}

	return rebootWin, err
}

func main() {

	app := cli.NewApp()
	app.Name = "CoreOS Update Logger"
	app.Version = "0.2.0"
	app.Compiled = time.Now()
	app.Usage = "Logs to ElasticSearch"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "indexname, i",
			Value:  "coreupdate",
			Usage:  "ElasticSearch indexName",
			EnvVar: "INDEX_NAME",
		},
		cli.StringFlag{
			Name:   "url, u",
			Usage:  "ElasticSearch endpoint",
			EnvVar: "URL",
		},
		cli.IntFlag{
			Name:   "freq, f",
			Value:  600,
			Usage:  "Frequency in seconds on when data is sent to ElasticSearch",
			EnvVar: "FREQ",
		},
		cli.StringFlag{
			Name:   "host",
			Usage:  "CoreOS Hostname or Ip",
			EnvVar: "HOST",
		},
		cli.StringFlag{
			Name:   "env, e",
			Usage:  "Environment tag (optional)",
			EnvVar: "ENV",
		},
		cli.StringFlag{
			Name:   "lock_smith, l",
			Value:  "20-cloudinit.conf",
			Usage:  "Location of locksmithd config /run/systemd/system/locksmithd.service.d/ (optional)",
			EnvVar: "LOCK_SMITH",
		},
		cli.StringFlag{
			Name:   "os_rel, o",
			Value:  "os-release",
			Usage:  "Location of /etc/os-release file (optional) ",
			EnvVar: "OS_REL",
		},
		cli.StringFlag{
			Name:   "update_conf, uc",
			Value:  "update.conf",
			Usage:  "Location of /etc/coreos/update.conf file (optional) ",
			EnvVar: "UPDATE_CONF",
		},
		cli.StringFlag{
			Name:   "uptime, up",
			Value:  "uptime",
			Usage:  "Location of /proc/uptime file (optional) ",
			EnvVar: "UPTIME",
		},
	}

	app.Action = func(c *cli.Context) error {
		if c.String("url") == "" || c.String("host") == "" {
			return cli.ShowAppHelp(c)
		}

		log.Println(color.YellowString("Starting Application"))

		log.Println("=Using Vars=")
		log.Println("indexname:", color.BlueString(c.String("indexname")))
		log.Println("url:", color.BlueString(c.String("url")))
		log.Println("freq:", color.BlueString(c.String("freq")))
		log.Println("host:", color.BlueString(c.String("host")))
		log.Println("env:", color.BlueString(c.String("env")))
		log.Println("lock_smith:", color.BlueString(c.String("lock_smith")))
		log.Println("os_rel:", color.BlueString(c.String("os_rel")))
		log.Println("update_conf:", color.BlueString(c.String("update_conf")))
		log.Println("uptime:", color.BlueString(c.String("uptime")))

		cfg, err := ini.LooseLoad(c.String("os_rel"), c.String("update_conf"))
		if err != nil {
			log.Fatal(err)
		}

		// if only doing reads speeds up operations
		cfg.BlockMode = false

		client, err := elastic.NewClient(elastic.SetURL(c.String("url")))
		if err != nil {
			log.Println(err)
		}

		// write to elastic search on a set interval
		for _ = range time.NewTicker(time.Duration(c.Int("freq")) * time.Second).C {
			if writeToES(client, c, cfg) != nil {
				log.Println(err)
			}
		}

		return nil
	}

	app.Run(os.Args)
}
